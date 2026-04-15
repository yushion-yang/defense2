#!/usr/bin/env python3
"""sync-i18n.py — 从配置 JSON 中提取显示文本，同步到 locale 文件。

用法:
    python3 scripts/sync-i18n.py          # 检查缺失（dry-run）
    python3 scripts/sync-i18n.py --write  # 写入更新

工作流:
1. 扫描 config/ 下的 JSON 配置文件，提取 label/name/description 等显示字段
2. 对比 config/i18n/zh.json，列出缺失的 key
3. --write 模式下自动添加缺失 key 到 zh.json（en.json 标记为 TODO）
"""

import json
import glob
import os
import sys
import re


def extract_ability_keys():
    """从 abilities.json 提取能力 label/display。"""
    keys = {}
    with open("config/towers/abilities.json") as f:
        abilities = json.load(f)
    for key, val in abilities.items():
        if not isinstance(val, dict) or "label" not in val:
            continue
        keys[f"ability.{key}.label"] = val["label"]
        if val.get("display"):
            keys[f"ability.{key}.display"] = val["display"]
    return keys


def extract_enemy_keys():
    """从 enemies-core.json 提取敌人 label。"""
    keys = {}
    with open("config/enemies/enemies-core.json") as f:
        enemies = json.load(f)
    if isinstance(enemies, list):
        for e in enemies:
            if "id" in e and "label" in e:
                keys[f"enemy.{e['id']}.label"] = e["label"]
    return keys


def extract_warden_keys():
    """从 wardens/defs/*.json 提取战灵显示文本。"""
    keys = {}
    fields = {
        "name": "name", "description": "description",
        "attackName": "attack_name", "attackDesc": "attack_desc",
        "specialName": "special_name", "specialDesc": "special_desc",
        "strengthDesc": "strength_desc", "counterTip": "counter_tip",
    }
    for wf in sorted(glob.glob("config/wardens/defs/*.json")):
        with open(wf) as f:
            w = json.load(f)
        wkey = os.path.basename(wf).replace(".json", "")
        for json_field, i18n_suffix in fields.items():
            if w.get(json_field):
                keys[f"warden.{wkey}.{i18n_suffix}"] = w[json_field]
    return keys


def extract_tower_keys():
    """从 towers.json 提取塔显示文本。"""
    keys = {}
    with open("config/towers/towers.json") as f:
        towers = json.load(f)
    for key, val in towers.items():
        if key.startswith("_") or not isinstance(val, dict):
            continue
        if val.get("label"):
            keys[f"tower.{key}.label"] = val["label"]
        if val.get("shortLabel"):
            keys[f"tower.{key}.short"] = val["shortLabel"]
        if val.get("description"):
            keys[f"tower.{key}.desc"] = val["description"]
    return keys


def extract_item_keys():
    """从 balance.json items 区段提取道具显示文本。"""
    keys = {}
    with open("config/balance.json") as f:
        bal = json.load(f)
    for item in bal.get("items", []):
        if "kind" not in item:
            continue
        if item.get("label"):
            keys[f"item.{item['kind']}.label"] = item["label"]
        if item.get("description"):
            keys[f"item.{item['kind']}.desc"] = item["description"]
    return keys


def extract_map_keys():
    """从 levels/map_*.json 提取地图名称。"""
    keys = {}
    for mf in sorted(glob.glob("config/levels/map_*.json")):
        with open(mf) as f:
            m = json.load(f)
        mid = os.path.basename(mf).replace(".json", "")
        if m.get("name"):
            keys[f"map.{mid}.name"] = m["name"]
        if m.get("description"):
            keys[f"map.{mid}.desc"] = m["description"]
    return keys


def main():
    write_mode = "--write" in sys.argv

    # 收集所有配置 key
    all_keys = {}
    all_keys.update(extract_ability_keys())
    all_keys.update(extract_enemy_keys())
    all_keys.update(extract_warden_keys())
    all_keys.update(extract_tower_keys())
    all_keys.update(extract_item_keys())
    all_keys.update(extract_map_keys())

    # 加载现有 locale 文件
    with open("config/i18n/zh.json") as f:
        zh = json.load(f)
    with open("config/i18n/en.json") as f:
        en = json.load(f)

    # 找缺失
    missing_zh = {k: v for k, v in all_keys.items() if k not in zh}
    missing_en = {k: v for k, v in all_keys.items() if k not in en}

    # 找 zh/en 不一致
    zh_only = [k for k in zh if not k.startswith("_") and k not in en]
    en_only = [k for k in en if not k.startswith("_") and k not in zh]

    if not missing_zh and not missing_en and not zh_only and not en_only:
        print(f"OK: all {len(all_keys)} config keys present in both locale files ({len(zh)} total keys)")
        return

    if missing_zh:
        print(f"\n{len(missing_zh)} key(s) missing from zh.json:")
        for k in sorted(missing_zh):
            print(f"  + {k}: {missing_zh[k]!r}")

    if missing_en:
        print(f"\n{len(missing_en)} key(s) missing from en.json:")
        for k in sorted(missing_en):
            print(f"  + {k}: [TODO: translate] {missing_en[k]!r}")

    if zh_only:
        print(f"\n{len(zh_only)} key(s) in zh.json but not en.json:")
        for k in sorted(zh_only):
            print(f"  ! {k}")

    if en_only:
        print(f"\n{len(en_only)} key(s) in en.json but not zh.json:")
        for k in sorted(en_only):
            print(f"  ! {k}")

    if write_mode:
        for k, v in missing_zh.items():
            zh[k] = v
        for k, v in missing_en.items():
            en[k] = f"[TODO] {v}"  # 标记待翻译

        with open("config/i18n/zh.json", "w") as f:
            json.dump(zh, f, ensure_ascii=False, indent=2, sort_keys=True)
            f.write("\n")
        with open("config/i18n/en.json", "w") as f:
            json.dump(en, f, ensure_ascii=False, indent=2, sort_keys=True)
            f.write("\n")
        print(f"\nWritten: zh.json ({len(zh)} keys), en.json ({len(en)} keys)")
    else:
        print(f"\nRun with --write to auto-fix. ({len(missing_zh)} zh + {len(missing_en)} en missing)")


if __name__ == "__main__":
    main()
