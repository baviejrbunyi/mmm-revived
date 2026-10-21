import argparse
import collections
import hashlib
import json
import sys
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))

from client_modules import REPO

RESOURCES = 'resources'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def environment(unity, raw):
    offset = raw.find(b'UnityFS\0', 8)
    if offset < 0:
        offset = raw.find(b'UnityFS\0')
    assert offset >= 0
    return unity.load(raw[offset:]), offset


def objects(env):
    return {obj.path_id: obj for obj in env.objects}


def strings(value, path=()):
    if isinstance(value, str):
        yield path, value
    elif isinstance(value, dict):
        for key, child in value.items():
            yield from strings(child, path + (key,))
    elif isinstance(value, list):
        for index, child in enumerate(value):
            yield from strings(child, path + (index,))


def without_strings(value):
    if isinstance(value, str):
        return None
    if isinstance(value, dict):
        return {key: without_strings(child) for key, child in value.items()}
    if isinstance(value, list):
        return [without_strings(child) for child in value]
    return value


def get(tree, path):
    for key in path:
        tree = tree[key]
    return tree


def put(tree, path, value):
    for key in path[:-1]:
        tree = tree[key]
    tree[path[-1]] = value


def translations(unity, original, english):
    old, _ = environment(unity, original)
    new, _ = environment(unity, english)
    before, after = objects(old), objects(new)
    if before.keys() != after.keys():
        return None
    changes = {}
    for path_id, obj in before.items():
        if obj.get_raw_data() == after[path_id].get_raw_data():
            continue
        if obj.type.name != 'MonoBehaviour':
            return None
        left, right = obj.read_typetree(), after[path_id].read_typetree()
        if without_strings(left) != without_strings(right):
            return None
        fields = {path: (value, get(right, path)) for path, value in strings(left) if get(right, path) != value}
        if fields:
            changes[path_id] = fields
    return changes


def apply(unity, target, changes, dictionary, keys):
    env, offset = environment(unity, target)
    current = objects(env)
    before = {path_id: obj.get_raw_data() for path_id, obj in current.items()}
    counts = collections.Counter()
    missing = []
    for path_id, obj in current.items():
        if obj.type.name != 'MonoBehaviour':
            continue
        tree = obj.read_typetree()
        changed = False
        expected = changes.get(path_id, {})
        for path, value in strings(tree):
            if path in expected and expected[path][0] == value:
                put(tree, path, expected[path][1])
                counts['object'] += 1
                changed = True
            elif path and path[-1] in keys and value in dictionary:
                put(tree, path, dictionary[value])
                counts['dictionary'] += 1
                changed = True
            elif path in expected:
                missing.append({'object': path_id, 'field': '.'.join(map(str, path)), 'source': value, 'previous': expected[path][0]})
        if changed:
            obj.set_raw_data(obj.save_typetree(tree))
    bundle = next(iter(env.files.values()))
    packed = target[:offset] + bundle.save(packer='original')
    verify, _ = environment(unity, packed)
    after = {obj.path_id: obj.get_raw_data() for obj in verify.objects}
    assert after.keys() == before.keys()
    for path_id, raw in after.items():
        if raw != before[path_id]:
            assert current[path_id].type.name == 'MonoBehaviour'
    return packed, counts, missing


def main():
    parser = argparse.ArgumentParser(description='Rebase the English UI bundles onto another client resource version.')
    parser.add_argument('--repo', type=Path, default=REPO)
    parser.add_argument('--previous', type=Path, required=True, help='folder with the original bundles the current English bundles were built from')
    parser.add_argument('--target', type=Path, nargs='+', required=True, help='folders with the target client bundles, searched in order')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder with UnityPy')
    parser.add_argument('--report', type=Path, help='write per-bundle results to this JSON file')
    parser.add_argument('--check', action='store_true', help='verify without writing assets')
    arguments = parser.parse_args()
    sys.path.insert(0, str(arguments.toolchain / 'tools/unitypy'))
    import UnityPy
    UnityPy.config.FALLBACK_UNITY_VERSION = '2021.3.41f1'
    folder = arguments.repo / 'assets' / RESOURCES
    bundles = sorted(path for path in folder.rglob('*.pck'))
    plans = {}
    dictionary_pairs = collections.defaultdict(set)
    keys = set()
    for path in bundles:
        relative = path.relative_to(folder).as_posix()
        original = (arguments.previous / relative).read_bytes()
        target = next((base / relative).read_bytes() for base in arguments.target if (base / relative).exists())
        english = path.read_bytes()
        changes = translations(UnityPy, original, english)
        plans[relative] = (original, target, english, changes)
        for fields in (changes or {}).values():
            for field, (source, translated) in fields.items():
                dictionary_pairs[source].add(translated)
                keys.add(field[-1])
    dictionary = {source: next(iter(values)) for source, values in dictionary_pairs.items() if len(values) == 1}
    report, outputs = {}, {}
    for relative, (original, target, english, changes) in plans.items():
        if target == original:
            report[relative] = {'status': 'unchanged upstream'}
            continue
        if changes is None:
            report[relative] = {'status': 'unsupported changes; rebuild required'}
            continue
        packed, counts, missing = apply(UnityPy, target, changes, dictionary, keys)
        report[relative] = {'status': 'rebased', 'translated': dict(counts), 'untranslated': missing, 'sha256': sha(packed)}
        outputs[folder / relative] = packed
    if arguments.report:
        arguments.report.write_text(json.dumps(report, ensure_ascii=False, indent=1), encoding='utf8')
    if not arguments.check:
        for path, data in outputs.items():
            path.write_bytes(data)
    print(json.dumps({name: {key: (len(value) if isinstance(value, list) else value) for key, value in entry.items() if key != 'sha256'} for name, entry in report.items()}, ensure_ascii=False, indent=1))
    if any(entry['status'].startswith('unsupported') for entry in report.values()):
        raise SystemExit(1)


if __name__ == '__main__':
    main()
