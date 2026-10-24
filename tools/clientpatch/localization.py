import argparse
import collections
import copy
import hashlib
import json
import re
import sys
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))

from client_modules import REPO, Toolchain, literal, parse, rebuild_constants, serialize

LOCALIZATION = 'localization'
PATCH_STRINGS = 'PatchStringsData'
CJK = re.compile('[\\u3400-\\u9fff]')
PERMITTED = {'LOADBOOL', 'LOADK', 'LOADKX', 'LOADNIL', 'NEWTABLE', 'RETURN', 'SETLIST', 'SETTABLE', 'EXTRAARG', 'MOVE'}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def flatten(value, path=()):
    if isinstance(value, dict):
        for key, child in value.items():
            yield from flatten(child, path + (key,))
    else:
        yield path, value


def index_of(folder):
    return {row['name']: row for row in json.loads((folder / 'installed/index.json').read_text())}


def assign(target, path, value):
    node = target
    for key in path[:-1]:
        node = node[key]
    node[path[-1]] = value


def unambiguous(pairs):
    sources = collections.defaultdict(set)
    for source, target in pairs:
        sources[source].add(target)
    return {source: next(iter(targets)) for source, targets in sources.items() if len(targets) == 1 and source not in targets}


def text_pairs(old_cn, old_en):
    return [(value, old_en[path]) for path, value in old_cn.items() if isinstance(value, str) and isinstance(old_en.get(path), str)]


def localize(tools, name, old_cn, old_en, target_original, english, shared):
    target = tools.module_data(target_original, name)
    new_cn = dict(flatten(target))
    by_path = {path: old_en[path] for path, value in old_cn.items() if isinstance(value, str) and path in old_en and old_en[path] != value}
    local = unambiguous(text_pairs(old_cn, old_en))
    english_flat = dict(flatten(english)) if english is not None else {}
    expected = copy.deepcopy(target)
    counts = collections.Counter()
    pending = []
    for path, value in new_cn.items():
        if not isinstance(value, str) or not CJK.search(value):
            continue
        candidate = english_flat.get(path)
        if old_cn.get(path) == value and path in by_path:
            translation, kind = by_path[path], 'path'
        elif value in local:
            translation, kind = local[value], 'source'
        elif isinstance(candidate, str) and candidate and not CJK.search(candidate):
            translation, kind = candidate, 'english_table'
        elif value in shared:
            translation, kind = shared[value], 'shared'
        else:
            counts['untranslated'] += 1
            pending.append({'path': [str(part) for part in path], 'source': value})
            continue
        assign(expected, path, translation)
        counts[kind] += 1
    expected_flat = dict(flatten(expected))
    replacements = unambiguous((value, expected_flat[path]) for path, value in new_cn.items() if isinstance(value, str))
    patched, changed = rebuild_constants(target_original, replacements)
    method = 'constants'
    if tools.module_data(patched, name) != expected:
        header, root = parse(target_original)
        ctor = root['children'][tools.methods(root)[(name, 'Ctor')][0]]
        assert {tools.opcodes[x & 63] for x in ctor['code']} <= PERMITTED, name
        patched, _ = tools.rebuild(target_original, {(name, 'Ctor'): 'function ' + name + ':Ctor()\nself.data=' + literal(expected) + '\nend\n'})
        method = 'constructor'
    observed = tools.module_data(patched, name)
    assert observed == expected, name
    observed_flat = dict(flatten(observed))
    assert observed_flat.keys() == new_cn.keys(), name
    for path, value in new_cn.items():
        if observed_flat[path] != value:
            assert isinstance(value, str) and isinstance(observed_flat[path], str), (name, path)
    details = {'method': method, 'changed_constants': changed if method == 'constants' else None, 'original_sha256': sha(target_original), 'patched_sha256': sha(patched), **counts}
    return patched, details, pending


def patch_strings(tools, original):
    expected = copy.deepcopy(tools.module_data(original, PATCH_STRINGS))
    count = 0
    for row in expected.values():
        if isinstance(row, dict) and row.get('en') and 'cn' in row:
            row['cn'] = row['en']
            count += 1
    patched = tools.compile_source(('return ' + literal(expected)).encode(), '@' + PATCH_STRINGS + '.lua')
    assert tools.module_data(patched, PATCH_STRINGS) == expected
    return patched, {'method': 'compiled', 'rows': count, 'original_sha256': sha(original), 'patched_sha256': sha(patched)}, []


def english_data(tools, english, name):
    if english is None or not name.endswith('_cn'):
        return None
    folder, rows = english
    row = rows.get(name[:-3] + '_en.lua')
    if row is None:
        return None
    try:
        return tools.module_data((folder / row['file']).read_bytes(), name[:-3] + '_en')
    except RuntimeError:
        return None


def main():
    parser = argparse.ArgumentParser(description='Rebase the English client data tables onto another client resource version.')
    parser.add_argument('--repo', type=Path, default=REPO)
    parser.add_argument('--previous', type=Path, required=True, help='analysis folder of the client resources the current English tables were built from')
    parser.add_argument('--analysis', type=Path, required=True, help='analysis folder of the target client resources')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder')
    parser.add_argument('--english', type=Path, help='folder with an index and dumped modules of an English client for strings without a previous translation')
    parser.add_argument('--add', nargs='*', default=[], help='additional target tables to translate')
    parser.add_argument('--rename', nargs='*', default=[], help='old=new table names for tables renamed in the target resources')
    parser.add_argument('--report', type=Path, help='write untranslated strings and per-table counts to this JSON file')
    parser.add_argument('--check', action='store_true', help='verify without writing assets')
    arguments = parser.parse_args()
    folder = arguments.repo / 'assets' / LOCALIZATION
    previous, target = index_of(arguments.previous), index_of(arguments.analysis)
    english = None
    if arguments.english is not None:
        english = (arguments.english, {row['name']: row for row in json.loads((arguments.english / 'index.json').read_text())})
    tools = Toolchain(arguments.toolchain)
    report, untranslated, outputs = {}, {}, {}
    try:
        existing = {}
        pairs = []
        for module in sorted(folder.glob('*.luac')):
            name = module.stem
            if name == PATCH_STRINGS:
                existing[name] = (module, None, None)
                continue
            old_cn = dict(flatten(tools.module_data((arguments.previous / 'installed' / previous[name + '.lua']['file']).read_bytes(), name)))
            old_en = dict(flatten(tools.module_data(module.read_bytes(), name)))
            existing[name] = (module, old_cn, old_en)
            pairs.extend(text_pairs(old_cn, old_en))
        shared = unambiguous(pairs)
        for name in arguments.add:
            assert name not in existing and name + '.lua' in target, name
            existing[name] = (folder / (name + '.luac'), {}, {})
        renames = dict(item.split('=', 1) for item in arguments.rename)
        for name, (module, old_cn, old_en) in sorted(existing.items()):
            entry = name + '.lua'
            output = module
            if entry not in target:
                if name not in renames:
                    report[name] = {'method': 'removed'}
                    outputs[module] = None
                    continue
                outputs[module] = None
                entry = renames[name] + '.lua'
                output = folder / (renames[name] + '.luac')
                assert entry in target and renames[name] not in existing, name
            target_name = entry[:-4]
            target_original = (arguments.analysis / 'installed' / target[entry]['file']).read_bytes()
            if name == PATCH_STRINGS:
                patched, details, pending = patch_strings(tools, target_original)
            else:
                table = english_data(tools, english, name)
                patched, details, pending = localize(tools, target_name, old_cn, old_en, target_original, table, shared)
                details['english_available'] = table is not None
            if output != module:
                details['renamed_from'] = name
            report[target_name] = details
            if pending:
                untranslated[target_name] = pending
            outputs[output] = patched
    finally:
        tools.close()
    totals = collections.Counter()
    for details in report.values():
        for key in ('path', 'source', 'english_table', 'shared', 'untranslated'):
            totals[key] += details.get(key, 0)
    if arguments.report:
        arguments.report.write_text(json.dumps({'tables': report, 'totals': totals, 'untranslated': untranslated}, ensure_ascii=False, indent=1), encoding='utf8')
    changed = sum(not module.exists() or module.read_bytes() != data for module, data in outputs.items() if data is not None)
    removed = sorted(module.name for module, data in outputs.items() if data is None and module not in [path for path, value in outputs.items() if value is not None])
    if not arguments.check:
        for module, data in outputs.items():
            if data is None:
                module.unlink()
            else:
                module.write_bytes(data)
    print(json.dumps({'tables': len(report), 'totals': totals, 'changed': changed, 'removed': removed}, indent=1))


if __name__ == '__main__':
    main()
