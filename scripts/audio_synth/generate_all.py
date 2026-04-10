#!/usr/bin/env python3
"""Generate all game audio: 3 BGM + 10 core SFX."""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from songs.menu import create_menu_bgm
from songs.battle import create_battle_bgm
from songs.boss import create_boss_bgm
from sfx_gen import generate_all_sfx

AUDIO_DIR = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'audio')


def main():
    print('=== Audio Generator ===\n')

    print('[1/4] Menu BGM...')
    create_menu_bgm().save(os.path.join(AUDIO_DIR, 'bgm-menu.wav'))
    print('  bgm-menu.wav OK')

    print('[2/4] Battle BGM...')
    create_battle_bgm().save(os.path.join(AUDIO_DIR, 'bgm-battle.wav'))
    print('  bgm-battle.wav OK')

    print('[3/4] Boss BGM...')
    create_boss_bgm().save(os.path.join(AUDIO_DIR, 'bgm-boss.wav'))
    print('  bgm-boss.wav OK')

    print('[4/4] Core SFX...')
    generate_all_sfx(AUDIO_DIR)

    print('\n=== All audio generated! ===')


if __name__ == '__main__':
    main()
