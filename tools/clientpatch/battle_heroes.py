import argparse
import json
import sys
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))

from client_modules import REPO, Toolchain

TABLES = ['battle_mode_new_data', 'player_temp_hero_data', 'player_temp_lv_data', 'character_data_new', 'character_level_data', 'artifact_data']
MAX_DYNAMIC_LEVEL = 80
SKIP_DEPLOY = 6
SKIP_DEPLOY_MANUAL = 16


def lua_list(value):
    if isinstance(value, dict):
        return [value[key] for key in sorted(value) if isinstance(key, int)]
    return value or []


def in_range(ranges, level):
    for key, value in (ranges or {}).items():
        bounds = json.loads(key) if isinstance(key, str) else key
        if isinstance(bounds, dict):
            bounds = [bounds[index] for index in sorted(bounds, key=int)]
        if len(bounds) == 2 and int(bounds[0]) <= level <= int(bounds[1]):
            return value
    return None


def hero_state(template, character, level, tables):
    row = (tables['player_temp_lv_data'].get(template['type']) or {}).get(level)
    if row is None:
        raise KeyError(f"player_temp_lv_data {template['type']} level {level}")
    three_star = character.get('star', 0) >= 3
    awakes = lua_list(template.get('awake'))[:row.get('awake_3star' if three_star else 'awake', 0)]
    talents = lua_list(row.get('talent'))
    state = {
        'level': level,
        'break': (tables['character_level_data'].get(level) or {}).get('lv2break', 0),
        'star': max(character.get('star', 1), row.get('star', 1)),
        'talents': {str(index): rank for index, rank in enumerate(talents, 1)},
        'awake': awakes,
        'artifacts': [],
    }
    for artifact, sub in sorted((in_range(template.get('arti'), level) or {}).items()):
        config = tables['artifact_data'][int(artifact)]
        state['artifacts'].append({'id': int(artifact), 'level': row.get('arti_lv', 0), 'template': lua_list(config.get('template'))[0], 'sub': {str(key): value for key, value in (sub or {}).items()}})
    highlight = in_range(template.get('hl'), level)
    if highlight:
        state['highlight'] = {'id': highlight, 'level': row.get('hl_lv', 1), 'awake': row.get('hl_awake_3star' if three_star else 'hl_awake', 0)}
    return state


def template_hero(template, character, base, dynamic, tables):
    if dynamic:
        return {'level': base, 'dynamic': True, 'levels': {str(level): hero_state(template, character, level, tables) for level in range(1, MAX_DYNAMIC_LEVEL + 1)}}
    return {'level': base, 'state': hero_state(template, character, min(max(base, 1), MAX_DYNAMIC_LEVEL), tables)}


def build(arguments):
    tools = Toolchain(arguments.toolchain)
    index = {row['name']: row for row in json.loads((arguments.analysis / 'installed/index.json').read_text())}
    try:
        data = {name: tools.table_data((arguments.analysis / 'installed' / index[name + '.lua']['file']).read_bytes(), name + '.lua') for name in TABLES}
    finally:
        tools.close()
    battles, templates, characters = data['battle_mode_new_data'], data['player_temp_hero_data'], data['character_data_new']
    path = arguments.repo / 'assets/datatables/rules.json'
    raw = path.read_text(encoding='utf-8')
    rules = json.loads(raw)
    trailer = raw[len(raw.rstrip('\n')):]
    assert json.dumps(rules, ensure_ascii=False, separators=(',', ':')) + trailer == raw
    settings = rules['battle_settings']
    report = {'battles': 0, 'skip_deploy': 0, 'dynamic': [], 'missing': []}
    for battle, config in battles.items():
        key = str(battle)
        entry = settings.get(key)
        if entry is None:
            continue
        entry.pop('template_heroes', None)
        entry.pop('skip_deploy', None)
        switches = config.get('switches') or {}
        if switches.get(SKIP_DEPLOY) and not switches.get(SKIP_DEPLOY_MANUAL):
            entry['skip_deploy'] = True
            report['skip_deploy'] += 1
        fixed = lua_list(config.get('hero'))
        if not fixed:
            continue
        assert entry['fixed_heroes'] == fixed, battle
        dynamic = bool((config.get('dynamicLv') or {}).get('hero_lv'))
        base = config.get('heroLv')
        base = 1 if base is None else base
        heroes = {}
        for template in fixed:
            row = templates.get(template)
            if row is None or row.get('hero') not in characters:
                report['missing'].append([battle, template])
                continue
            heroes[str(row['hero'])] = template_hero(row, characters[row['hero']], base, dynamic, data)
        if dynamic:
            report['dynamic'].append(battle)
        entry['template_heroes'] = heroes
        report['battles'] += 1
    output = json.dumps(rules, ensure_ascii=False, separators=(',', ':')) + trailer
    if not arguments.check:
        path.write_text(output, encoding='utf-8', newline='')
    report['changed'] = output != raw
    print(json.dumps(report))


def main():
    parser = argparse.ArgumentParser(description='Export fixed battle template heroes into assets/datatables/rules.json.')
    parser.add_argument('--repo', type=Path, default=REPO)
    parser.add_argument('--analysis', type=Path, required=True, help='client analysis folder with the installed index and bytecode')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder with xlua.offline.dll')
    parser.add_argument('--check', action='store_true', help='report without writing rules.json')
    build(parser.parse_args())


if __name__ == '__main__':
    main()
