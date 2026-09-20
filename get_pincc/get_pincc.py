#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
获取 Pincc 平台「我的 API 秘钥」中每个用户（key）当日用量，按用量从高到低排序，推送到飞书群。

数据链路（已通过前端逆向确认）：
  1. 鉴权：POST /api/v1/auth/refresh  body={refresh_token} -> 新 access_token（有效期 24h）
     （登录接口有 Turnstile 人机验证，脚本无法自动过；改用 refresh_token 绕开）
  2. 列表：GET /api/v1/keys?page=&page_size=  -> data.items[]，每条含 name / key(sk-) / group.name
  3. 用量：逐个 key 用其自身 sk- 调 GET /v1/usage，取 usage.today 的准确当日用量
       actual_cost -> 乘分组倍率后的实际扣费（展示用这个）
       cost        -> 原价
  4. 推送：飞书自定义机器人 Webhook（可选签名校验）

【如何获取 refresh_token（一次性操作，需在有浏览器的机器上做一次）】
  1. 浏览器登录 https://jp.pincc.ai
  2. F12 -> Application -> Local Storage -> https://jp.pincc.ai
  3. 复制 refresh_token 的值
  4. 写入 PINCC_TOKEN_FILE（默认见下），或 export PINCC_REFRESH_TOKEN=...
  之后脚本每次运行会自动刷新并把新 refresh_token 写回该文件，长期自动续期。

配置（环境变量优先）：
  export PINCC_REFRESH_TOKEN=xxxx             # 或写入 token 文件
  export FEISHU_WEBHOOK=https://open.feishu.cn/open-apis/bot/v2/hook/xxxx
  export FEISHU_SECRET=xxxx                   # 可选：机器人开了「签名校验」时填
  export PINCC_BASE_URL=https://jp.pincc.ai   # 可选，默认 jp.pincc.ai
  export PINCC_TOKEN_FILE=...                  # 可选，持久化 refresh_token 的文件

用法：
  python3 get_pincc.py                 # 统计每个用户今日用量，前 30 名，推送飞书
  python3 get_pincc.py --dry-run       # 只打印不推送（首次调试用）
  python3 get_pincc.py --top 50        # 取前 10 名
  python3 get_pincc.py --all           # 连 0 用量的用户也显示
"""

import os
import sys
import time
import hmac
import base64
import hashlib
import argparse
import datetime as dt
from concurrent.futures import ThreadPoolExecutor, as_completed

try:
    import requests
except ImportError:
    print("缺少 requests 库，请先运行：pip3 install requests")
    sys.exit(1)

# ------------------------- 配置 -------------------------
BASE_URL       = os.getenv("PINCC_BASE_URL", "https://jp.pincc.ai").rstrip("/")
FEISHU_WEBHOOK = os.getenv("FEISHU_WEBHOOK", "https://open.feishu.cn/open-apis/bot/v2/hook/cf86a7db-775e-4f06-8022-84e722d31cf3")
FEISHU_SECRET  = os.getenv("FEISHU_SECRET", "")
TOKEN_FILE     = os.path.expanduser(os.getenv(
    "PINCC_TOKEN_FILE", "/Users/alada/Documents/scripts/get_pincc/pincc_token"))

API = BASE_URL + "/api/v1"
TIMEOUT = 30
MAX_WORKERS = 10          # 逐 key 查用量的并发数


# ------------------------- 鉴权（refresh_token 方案）-------------------------
def _load_refresh_token():
    t = os.getenv("PINCC_REFRESH_TOKEN", "").strip()
    if t:
        return t
    if os.path.exists(TOKEN_FILE):
        with open(TOKEN_FILE) as f:
            return f.read().strip()
    return ""


def _save_refresh_token(token):
    with open(TOKEN_FILE, "w") as f:
        f.write(token)


def get_access_token(session):
    """用 refresh_token 换取 access_token，并持久化轮换后的新 refresh_token。"""
    refresh_token = _load_refresh_token()
    if not refresh_token:
        _die("未找到 refresh_token。请从浏览器 Local Storage 复制 refresh_token 写入 "
             + TOKEN_FILE + " ，或 export PINCC_REFRESH_TOKEN=...")

    r = _req_with_retry(session, "POST", f"{API}/auth/refresh",
                        json={"refresh_token": refresh_token})
    data = _json(r, "刷新 token")
    access = data.get("access_token") or data.get("data", {}).get("access_token")
    if not access:
        _die("refresh 未返回 access_token（token 可能已过期，请重新从浏览器获取）", data)
    new_refresh = data.get("refresh_token") or data.get("data", {}).get("refresh_token")
    if new_refresh:
        _save_refresh_token(new_refresh)
    return access


# ------------------------- 拉取 keys 列表 -------------------------
def get_keys(session, token):
    """分页拉取 /keys，返回全部 key（含各人当日用量）。"""
    headers = {"Authorization": f"Bearer {token}"}
    items, page, page_size = [], 1, 100
    while True:
        r = _req_with_retry(session, "GET", f"{API}/keys",
                            headers=headers,
                            params={"page": page, "page_size": page_size})
        if r.status_code >= 400:
            _die(f"获取 keys 失败 (HTTP {r.status_code})", r.text[:300])
        data = _json(r, "获取 keys").get("data", {})
        batch = data.get("items", [])
        items.extend(batch)
        pages = data.get("pages") or 1
        if page >= pages or not batch:
            break
        page += 1
    return items


# ------------------------- 逐 key 查准确的今日用量 -------------------------
def _key_today_usage(session, key_obj):
    """用某个 key 自己（Bearer sk-）调 /v1/usage，取 usage.today 的准确当日用量。

    对瞬时失败（502/SSL/空响应/缺字段）多次重试；彻底失败不静默回退到不准的
    usage_1d，而是标记 ok=False，交由上层展示时提醒。
    """
    sk = key_obj.get("key")
    result = {
        "name": key_obj.get("name") or f"key_{key_obj.get('id', '?')}",
        "group": (key_obj.get("group") or {}).get("name") or "-",
        "actual_cost": 0.0, "cost": 0.0, "tokens": 0, "requests": 0,
        "ok": False,
    }
    if not sk:
        return result

    last_err = ""
    for attempt in range(1, 7):                       # 最多 6 次
        try:
            r = session.get(f"{BASE_URL}/v1/usage",
                            headers={"Authorization": f"Bearer {sk}"}, timeout=TIMEOUT)
            if r.status_code in (502, 503, 504):
                last_err = f"HTTP {r.status_code}"
            else:
                today = (r.json().get("usage") or {}).get("today")
                if today is None:
                    last_err = "响应缺 usage.today"
                else:
                    result.update(
                        actual_cost=_f(today.get("actual_cost")),
                        cost=_f(today.get("cost")),
                        tokens=int(_f(today.get("total_tokens"))),
                        requests=int(_f(today.get("requests"))),
                        ok=True,
                    )
                    return result
        except Exception as e:
            last_err = type(e).__name__
        time.sleep(1.5 * attempt)                     # 递增退避

    print(f"  ⚠️ {result['name']} 查用量失败（{last_err}），该行标记为不确定")
    result["actual_cost"] = _f(key_obj.get("usage_1d"))   # 兜底近似值，但 ok=False
    return result


def fetch_today_usage(session, keys):
    """并发查询所有 key 的今日用量。"""
    results = []
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        futs = {pool.submit(_key_today_usage, session, k): k for k in keys}
        done = 0
        for fut in as_completed(futs):
            results.append(fut.result())
            done += 1
            if done % 20 == 0 or done == len(keys):
                print(f"  … 已查询 {done}/{len(keys)}")
    return results


def _req_with_retry(session, method, url, tries=5, **kw):
    """对 502/503/504 网关错误及网络异常自动重试，间隔递增（5,10,15,20…秒）。"""
    r = None
    for attempt in range(1, tries + 1):
        try:
            r = session.request(method, url, timeout=TIMEOUT, **kw)
            if r.status_code not in (502, 503, 504):
                return r
            reason = f"网关 {r.status_code}"
        except requests.RequestException as e:
            reason = f"网络异常 {type(e).__name__}"
        if attempt < tries:
            wait = 5 * attempt
            print(f"  ⏳ {reason}，{wait}s 后第 {attempt + 1}/{tries} 次重试 …")
            time.sleep(wait)
    if r is None:
        _die(f"请求 {url} 连续 {tries} 次失败（网络异常或服务持续不可用）")
    return r


# ------------------------- 组装并推送飞书 -------------------------
def _f(v):
    return float(v) if isinstance(v, (int, float)) else 0.0


def build_message(usages, top, show_all):
    """返回 (title, summary, rows)。rows 每项 = (用户列文本, 分组, 金额文本)。"""
    rows = sorted(usages, key=lambda u: u["actual_cost"], reverse=True)
    if not show_all:
        rows = [u for u in rows if u["actual_cost"] > 0]
    if top > 0:
        rows = rows[:top]

    total = sum(u["actual_cost"] for u in usages)
    active = sum(1 for u in usages if u["actual_cost"] > 0)

    table = []
    for i, u in enumerate(rows, 1):
        medal = "🥇🥈🥉"[i - 1] if i <= 3 else f"{i}."
        amount = f"¥{u['actual_cost']:,.2f}"
        if not u.get("ok", True):
            amount += " ⚠️"          # 该值查询失败、为近似
        table.append((f"{medal} {u['name']}", u["group"], amount))

    today = dt.date.today().isoformat()
    title = f"📊 AI Coding 用户当日用量排行（{today}）"
    summary = (f"共 {len(usages)} 个用户，其中 {active} 人今日有消费，"
               f"合计 **¥{total:,.2f}**\n\n**📈 用量当日排名榜：**")
    return title, summary, table


def _col(header, values, weight):
    """构造飞书 column_set 的一列：顶部加粗表头 + 逐行值。"""
    content = f"**{header}**\n" + "\n".join(values) if values else f"**{header}**"
    return {
        "tag": "column", "width": "weighted", "weight": weight,
        "vertical_align": "top",
        "elements": [{"tag": "div", "text": {"tag": "lark_md", "content": content}}],
    }


def send_feishu(title, summary, table):
    if not FEISHU_WEBHOOK:
        print("未配置 FEISHU_WEBHOOK，跳过推送。")
        return

    elements = [{"tag": "div", "text": {"tag": "lark_md", "content": summary}},
                {"tag": "hr"}]
    if table:
        names = [r[0] for r in table]
        groups = [r[1] for r in table]
        amounts = [r[2] for r in table]
        elements.append({
            "tag": "column_set", "flex_mode": "none", "horizontal_spacing": "default",
            "columns": [
                _col("用户", names, 5),
                _col("分组", groups, 4),
                _col("金额", amounts, 3),
            ],
        })
    else:
        elements.append({"tag": "div", "text": {"tag": "lark_md", "content": "_今日暂无用量_"}})

    elements += [
        {"tag": "hr"},
        {"tag": "note", "elements": [
            {"tag": "lark_md",
             "content": f"数据来源 {BASE_URL} · 生成于 "
                        + dt.datetime.now().strftime("%Y-%m-%d %H:%M")}
        ]},
    ]

    payload = {
        "msg_type": "interactive",
        "card": {
            "config": {"wide_screen_mode": True},
            "header": {"title": {"tag": "plain_text", "content": title}, "template": "blue"},
            "elements": elements,
        },
    }
    if FEISHU_SECRET:
        ts = str(int(time.time()))
        payload["timestamp"] = ts
        payload["sign"] = _feishu_sign(FEISHU_SECRET, ts)

    r = requests.post(FEISHU_WEBHOOK, json=payload, timeout=TIMEOUT)
    resp = r.json() if r.headers.get("content-type", "").startswith("application/json") else {}
    if r.status_code == 200 and resp.get("code") in (0, None):
        print("✅ 已推送到飞书群")
    else:
        print(f"⚠️ 飞书推送异常：HTTP {r.status_code} {r.text}")


def _feishu_sign(secret, timestamp):
    s = f"{timestamp}\n{secret}"
    return base64.b64encode(hmac.new(s.encode(), digestmod=hashlib.sha256).digest()).decode()


# ------------------------- 工具 -------------------------
def _json(resp, what):
    try:
        return resp.json()
    except Exception:
        _die(f"{what}：返回不是 JSON（HTTP {resp.status_code}）", resp.text[:300])


def _die(msg, detail=None):
    print(f"❌ {msg}")
    if detail is not None:
        print("   详情：", detail)
    sys.exit(1)


def main():
    ap = argparse.ArgumentParser(description="Pincc 用户当日用量排行 -> 飞书")
    ap.add_argument("--top", type=int, default=10, help="取前 N 名（默认 10，0=全部）")
    ap.add_argument("--all", action="store_true", help="连 0 用量的用户也显示")
    ap.add_argument("--dry-run", action="store_true", help="只打印不推送飞书")
    args = ap.parse_args()

    session = requests.Session()
    session.headers.update({
        "User-Agent": "Mozilla/5.0",
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Origin": BASE_URL,
        "Referer": BASE_URL + "/",
    })

    print(f"→ 刷新 token ({BASE_URL}) …")
    token = get_access_token(session)
    print("→ 鉴权成功，拉取 keys 列表 …")
    keys = get_keys(session, token)
    print(f"→ 拿到 {len(keys)} 个 key，逐个查询今日准确用量 …")
    usages = fetch_today_usage(session, keys)

    title, summary, table = build_message(usages, args.top, args.all)

    # 本地预览（对齐打印）
    print("\n" + title)
    print(summary.replace("**", ""))
    print(f"{'用户':<18}{'分组':<22}{'金额':>12}")
    for name, group, amount in table:
        print(f"{name:<18}{group:<22}{amount:>12}")
    print()

    if args.dry_run:
        print("(--dry-run 模式，未推送飞书)")
    else:
        send_feishu(title, summary, table)


if __name__ == "__main__":
    main()
