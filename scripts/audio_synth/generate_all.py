#!/usr/bin/env python3
"""Generate all game audio: 3 BGM (SF2) + 10 core SFX.

用法:
    python generate_all.py              # 生成全部（BGM + SFX）
    python generate_all.py --song menu  # 只生成 menu BGM
    python generate_all.py --song all   # 只生成全部 BGM
    python generate_all.py --sfx        # 只生成 SFX
    python generate_all.py --legacy     # 用旧的波形合成引擎生成 BGM
"""

import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

AUDIO_DIR = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'audio')


def generate_bgm(song_name: str | None = None, legacy: bool = False):
    """生成 BGM。song_name: menu/battle/boss/None(全部)。"""
    if legacy:
        from songs.menu_legacy import create_menu_bgm
        from songs.battle_legacy import create_battle_bgm
        from songs.boss_legacy import create_boss_bgm
        print('  [legacy mode: 波形合成]')
    else:
        from songs.menu import create_menu_bgm
        from songs.battle import create_battle_bgm
        from songs.boss import create_boss_bgm

    songs = {
        'menu': ('bgm-menu.wav', create_menu_bgm),
        'battle': ('bgm-battle.wav', create_battle_bgm),
        'boss': ('bgm-boss.wav', create_boss_bgm),
    }

    targets = [song_name] if song_name else list(songs.keys())
    for i, name in enumerate(targets):
        filename, factory = songs[name]
        print(f'  [{i+1}/{len(targets)}] {name} BGM...')
        factory().save(os.path.join(AUDIO_DIR, filename))


def main():
    parser = argparse.ArgumentParser(description='Game audio generator')
    parser.add_argument('--song', choices=['menu', 'battle', 'boss', 'all'],
                        help='Only generate specific BGM (or "all" for all BGMs)')
    parser.add_argument('--sfx', action='store_true', help='Only generate SFX')
    parser.add_argument('--legacy', action='store_true',
                        help='Use legacy waveform synthesis for BGM')
    args = parser.parse_args()

    print('=== Audio Generator ===\n')

    if args.sfx:
        print('[SFX]')
        from sfx_gen import generate_all_sfx
        generate_all_sfx(AUDIO_DIR)
    elif args.song:
        print('[BGM]')
        song = None if args.song == 'all' else args.song
        generate_bgm(song, legacy=args.legacy)
    else:
        print('[BGM]')
        generate_bgm(legacy=args.legacy)
        print('\n[SFX]')
        from sfx_gen import generate_all_sfx
        generate_all_sfx(AUDIO_DIR)

    print('\n=== Done! ===')


if __name__ == '__main__':
    main()
