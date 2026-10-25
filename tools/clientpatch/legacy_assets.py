import argparse
import hashlib
import io
import json
import re
import struct
import sys
import zipfile
import zlib
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))

from client_modules import REPO, Toolchain

RESOURCES = 'resources'
CATALOG = 'resources/legacy_assets.json'
SUFFIX = '_legacy'
TABLE = 'highlight_data_cn'
VERSIONS = 'version_control_data'
FIRST = re.compile(r'patch_first_[a-z]+_win32_([0-9.]+)$')
PREFIX = 16
SIGNATURE = b'UnityFS\0'


def read_assets(raw):
    count = struct.unpack_from('<I', raw, 0)[0]
    pos, nodes = 4, []
    for _ in range(count):
        size = raw[pos]
        name = raw[pos + 1:pos + 1 + size].decode('utf8')
        pos += 1 + size
        package = struct.unpack_from('<H', raw, pos)[0]
        pos += 2
        children = None
        if package == 0:
            total = struct.unpack_from('<I', raw, pos)[0]
            children = list(struct.unpack_from('<%dI' % total, raw, pos + 4))
            pos += 4 + 4 * total
        nodes.append((name, package, children))
    assert pos == len(raw)
    paths, stack = {}, [(0, '')]
    while stack:
        index, prefix = stack.pop()
        name, package, children = nodes[index]
        full = prefix + '/' + name if prefix else name
        if children is None:
            paths[full] = package
        else:
            stack.extend((child, full) for child in children)
    return paths


def read_packages(raw):
    count = struct.unpack_from('<I', raw, 0)[0]
    pos, packages = 4, []
    for _ in range(count):
        size = raw[pos]
        name = raw[pos + 1:pos + 1 + size].decode('utf8')
        pos += 1 + size
        head = raw[pos:pos + 8]
        total = struct.unpack_from('<H', raw, pos + 8)[0]
        deps = list(struct.unpack_from('<%dH' % total, raw, pos + 10))
        pos += 10 + 2 * total
        shaders = struct.unpack_from('<I', raw, pos)[0]
        pos += 4
        for _ in range(shaders):
            pos += 1 + raw[pos] + 4
        packages.append({'name': name, 'head': head, 'deps': deps})
    assert pos == len(raw)
    return packages


def first_package(content):
    with zipfile.ZipFile(content) as outer:
        candidates = []
        for name in outer.namelist():
            match = FIRST.search(name)
            if match:
                candidates.append((tuple(int(part) for part in match.group(1).split('.')), name))
        name = max(candidates)[1]
        with zipfile.ZipFile(io.BytesIO(outer.read(name))) as inner:
            return {entry: inner.read(entry) for entry in ('assets.bytes', 'packages.bytes', 'scripts.pck')}


def streaming(legacy):
    for path in [legacy, *legacy.glob('*_Data/StreamingAssets')]:
        if (path / 'assets.bytes').is_file() and (path / 'packages.bytes').is_file():
            return path
    raise SystemExit('legacy client StreamingAssets not found under %s' % legacy)


def prefix_length(name):
    encoded = name.encode()
    return PREFIX + (sum(encoded) + len(encoded)) % PREFIX


def wrap(name, marker, raw):
    return raw[:8] + marker + raw[12:prefix_length(name)] + raw


def check_prefixes(legacy):
    marker = None
    for path in sorted((legacy / 'common').glob('*.pck')):
        with path.open('rb') as handle:
            head = handle.read(64)
        name = 'common/' + path.stem
        assert head.find(SIGNATURE, 8) == prefix_length(name), name
        marker = marker or head[8:12]
        assert head[8:12] == marker, name
    return marker


def split_bundle(data):
    offset = data.find(SIGNATURE, 8)
    return (data[:offset], data[offset:]) if offset > 0 else (b'', data)


def script_tables(tools, scripts, names):
    _, bundle = split_bundle(scripts)
    tables = {}
    for obj in tools.unity.load(bundle).objects:
        if obj.type.name != 'TextAsset':
            continue
        tree = obj.read_typetree()
        name = tree['m_Name']
        if name[:-4] in names:
            tables[name[:-4]] = tools.module_data(tree['m_Script'].encode('utf-8', errors='surrogateescape'), name[:-4])
    assert set(tables) == set(names), names
    return tables


def wanted_assets(row, paths):
    keys = [str(row[field]) for field in ('icon', 'spine') if row.get(field)]
    found = []
    for path in paths:
        parts = path.split('/')
        stem = parts[-1].rsplit('.', 1)[0]
        if any(stem == key or key in parts[:-1] for key in keys):
            found.append(path)
    return sorted(found)


def cab_of(bundle):
    names = [name for name in bundle.files if not name.endswith(('.resS', '.resource'))]
    assert len(names) == 1, names
    return names[0]


def subset(tools, data, keep, name, marker):
    _, raw = split_bundle(data)
    env = tools.unity.load(raw)
    bundle = next(iter(env.files.values()))
    old_cab = cab_of(bundle)
    serialized = bundle.files[old_cab]
    holder = next(obj for obj in serialized.objects.values() if obj.type.name == 'AssetBundle')
    tree = holder.read_typetree()
    entries = [(path, info) for path, info in tree['m_Container'] if path in keep]
    assert {path for path, _ in entries} == keep, sorted(keep - {path for path, _ in entries})
    preload, container, retained = [], [], {holder.path_id}
    for path, info in entries:
        refs = tree['m_PreloadTable'][info['preloadIndex']:info['preloadIndex'] + info['preloadSize']]
        container.append((path, {'preloadIndex': len(preload), 'preloadSize': len(refs), 'asset': info['asset']}))
        preload.extend(refs)
        retained.add(info['asset']['m_PathID'])
        retained.update(ref['m_PathID'] for ref in refs if ref['m_FileID'] == 0)
    for path_id in [key for key in serialized.objects if key not in retained]:
        del serialized.objects[path_id]
    new_cab = 'CAB-' + hashlib.md5(name.encode()).hexdigest()
    streams = {key: value for key, value in bundle.files.items() if key != old_cab}
    for obj in list(serialized.objects.values()):
        if obj.type.name != 'Texture2D':
            continue
        texture = obj.read_typetree()
        stream = texture.get('m_StreamData') or {}
        if stream.get('path'):
            source = streams[stream['path'].rsplit('/', 1)[-1]]
            source.Position = stream['offset']
            texture['image data'] = source.read_bytes(stream['size'])
            texture['m_CompleteImageSize'] = stream['size']
            texture['m_StreamData'] = {'offset': 0, 'size': 0, 'path': ''}
            obj.save_typetree(texture)
    tree.update({'m_Name': name, 'm_AssetBundleName': name, 'm_Container': container, 'm_PreloadTable': preload})
    holder.save_typetree(tree)
    serialized.name = new_cab
    bundle.files = {new_cab: serialized}
    output = bundle.save(packer='lz4')
    check = next(iter(tools.unity.load(output).files.values()))
    assert list(check.files) == [new_cab]
    return wrap(name, marker, output), new_cab


def externals(tools, data):
    _, raw = split_bundle(data)
    bundle = next(iter(tools.unity.load(raw).files.values()))
    serialized = bundle.files[cab_of(bundle)]
    return cab_of(bundle), [item.path.rsplit('/', 1)[-1] for item in serialized.externals]


def build(arguments):
    legacy = streaming(arguments.legacy)
    marker = check_prefixes(legacy)
    upstream = first_package(arguments.content)
    current_assets = read_assets(zlib.decompress(upstream['assets.bytes']))
    current_packages = {package['name'] for package in read_packages(zlib.decompress(upstream['packages.bytes']))}
    old_assets = read_assets(zlib.decompress((legacy / 'assets.bytes').read_bytes()))
    old_packages = read_packages(zlib.decompress((legacy / 'packages.bytes').read_bytes()))
    tools = Toolchain(arguments.toolchain)
    try:
        current = script_tables(tools, upstream['scripts.pck'], [TABLE, VERSIONS])
        previous = script_tables(tools, (legacy / 'scripts.pck').read_bytes(), [TABLE])
        released = set(current[VERSIONS])
        fields, assets, packages, files = {}, {}, {}, {}
        for key, row in sorted(current[TABLE].items()):
            old = previous[TABLE].get(key)
            if row.get('version') in released or not old or old.get('version') not in released:
                continue
            missing = [path for path in wanted_assets(old, old_assets) if path not in current_assets]
            if not missing:
                continue
            fields[str(key)] = {'version': old['version']}
            for path in missing:
                assets[path] = old_packages[old_assets[path] - 1]['name']
        bundles = {}
        for path, package in assets.items():
            bundles.setdefault(package, set()).add(path.lower())
        names = {}
        queue = list(bundles)
        cabs = {}
        while queue:
            package = queue.pop()
            if package in names:
                continue
            record = next(item for item in old_packages if item['name'] == package)
            data = (legacy / (package + '.pck')).read_bytes()
            if package in current_packages and package in bundles:
                target = package + SUFFIX
                data, cab = subset(tools, data, bundles[package], target, marker)
            else:
                assert package not in current_packages, package
                target = package
            names[package] = target
            cab, refs = externals(tools, data)
            cabs[cab] = target
            files[target] = data
            packages[target] = {'head': record['head'].hex(), 'refs': refs, 'deps': [old_packages[index - 1]['name'] for index in record['deps']]}
            for dep in packages[target]['deps']:
                if dep not in current_packages and dep not in names:
                    queue.append(dep)
        for package in list(packages):
            deps = {dep for dep in packages[package]['deps'] if dep in current_packages}
            for ref in packages[package].pop('refs'):
                if ref in cabs and cabs[ref] != package:
                    deps.add(cabs[ref])
            packages[package]['deps'] = sorted(deps)
        assets = {path: names[package] for path, package in assets.items()}
    finally:
        tools.close()
    folder = arguments.repo / 'assets'
    for package, data in files.items():
        target = folder / RESOURCES / (package + '.pck')
        target.parent.mkdir(parents=True, exist_ok=True)
        if not arguments.check:
            target.write_bytes(data)
    catalog = {'packages': dict(sorted(packages.items())), 'assets': dict(sorted(assets.items())), 'fields': {TABLE: fields}}
    if not arguments.check:
        (folder / CATALOG).write_text(json.dumps(catalog, indent=2) + '\n', encoding='utf8')
    print(json.dumps({'packages': {name: len(data) for name, data in files.items()}, 'assets': len(assets), 'fields': fields}, indent=2))


def main():
    parser = argparse.ArgumentParser(description='Port client assets that the current resources dropped from an older client build.')
    parser.add_argument('--repo', type=Path, default=REPO)
    parser.add_argument('--legacy', type=Path, required=True, help='older client folder or its StreamingAssets folder')
    parser.add_argument('--content', type=Path, required=True, help='content zip with the current client resources')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder')
    parser.add_argument('--check', action='store_true', help='report without writing assets')
    build(parser.parse_args())


if __name__ == '__main__':
    main()
