import argparse
import ctypes as C
import hashlib
import json
import re
import shutil
import struct
import subprocess
import sys
import tempfile
from pathlib import Path

sys.dont_write_bytecode = True

HERE = Path(__file__).resolve().parent
SOURCES = HERE
REPO = HERE.parents[1]

MODULES = [
    ('PanelFriendTip.lua', 'scripts/PanelFriendTip.luac', {('PanelFriendTip', 'Show'): 'friendduel/PanelFriendTip.Show.lua'}),
    ('FriendInfo.lua', 'scripts/FriendInfo.luac', {('FriendItem', 'SetInfo'): 'friendduel/FriendInfo.SetInfo.lua'}),
    ('FriendComp.lua', 'scripts/FriendComp.luac', {('FriendComp', 'OnInit'): 'friendduel/FriendComp.OnInit.lua', ('FriendComp', 'onJoinWorldChannel'): 'worldchat/FriendComp.onJoinWorldChannel.lua', ('FriendComp', 'onListWorldChannel'): 'worldchat/FriendComp.onListWorldChannel.lua'}),
    ('BattleComp.lua', 'scripts/BattleComp.luac', {('BattleComp', 'PvPBattleStart'): 'realtimesync/BattleComp.PvPBattleStart.lua', ('BattleComp', 'PvPBattleMatched'): 'realtimesync/BattleComp.PvPBattleMatched.lua'}),
    ('Battle.lua', 'scripts/Battle.luac', {('Battle', 'CheckAutoPlayCard'): 'realtimesync/Battle.CheckAutoPlayCard.lua', ('Battle', 'CheckSwitch'): 'realtimesync/Battle.CheckSwitch.lua'}),
    ('LogicInterface.lua', 'scripts/LogicInterface.luac', {('LogicInterface', 'UseCard'): 'realtimesync/LogicInterface.UseCard.lua'}),
    ('AIEngine.lua', 'scripts/AIEngine.luac', {('AIEngine', 'TryUseCard'): 'realtimesync/AIEngine.TryUseCard.lua'}),
    ('BattleNewStage.lua', 'scripts/BattleNewStage.luac', {('BattleNewStage', '_TryDoFrame'): 'realtimesync/BattleNewStage.TryDoFrame.lua', ('BattleNewStage', '_GetPreloadPanels'): 'realtimesync/BattleNewStage._GetPreloadPanels.lua', ('BattleNewStage', 'ShowGuide'): 'tutorial/BattleNewStage.ShowGuide.lua'}),
    ('SceneTimeControlComp.lua', 'scripts/SceneTimeControlComp.luac', {('SceneTimeControlComp', 'StopScaledTime'): 'realtimesync/SceneTimeControlComp.StopScaledTime.lua', ('SceneTimeControlComp', '_InitAutoState'): 'realtimesync/SceneTimeControlComp._InitAutoState.lua', ('SceneTimeControlComp', '_InitSpeedUpState'): 'realtimesync/SceneTimeControlComp._InitSpeedUpState.lua'}),
    ('PanelBattleNew.lua', 'scripts/PanelBattleNew.luac', {('PanelBattleNew', 'SetCardForceNonInteractable'): 'realtimesync/PanelBattleNew.SetCardForceNonInteractable.lua', ('PanelBattleNew', 'SetSpeedUpBtnVisible'): 'realtimesync/PanelBattleNew.SetSpeedUpBtnVisible.lua'}),
    ('ResolutionTool.lua', 'scripts/ResolutionTool.luac', {('ResolutionTool', '_GetDeviceResolution'): 'window/ResolutionTool._GetDeviceResolution.lua', ('ResolutionTool', 'OnWindowResize'): 'window/ResolutionTool.OnWindowResize.lua'}),
    ('PanelSkipGuide.lua', 'scripts/PanelSkipGuide.luac', {('PanelSkipGuide', '_OnSkipClicked'): 'tutorial/PanelSkipGuide._OnSkipClicked.lua'}),
    ('HeroHighlight.lua', 'scripts/HeroHighlight.luac', {('HeroHighlight', '_OnHighlightModelClicked'): 'tutorial/HeroHighlight._OnHighlightModelClicked.lua'}),
    ('UniSDKHandler.lua', 'scripts/UniSDKHandler.luac', {('UniSDKHandler', 'CheckOrder'): 'freeshop/UniSDKHandler.CheckOrder.lua', ('UniSDKHandler', 'GetProductPrice'): 'freeshop/UniSDKHandler.GetProductPrice.lua'}),
    ('DrawComp.lua', 'scripts/DrawComp.luac', {('DrawComp', 'OnInit'): 'economy/DrawComp.OnInit.lua'}),
    ('PanelLimitDrawUp.lua', 'scripts/PanelLimitDrawUp.luac', {('PanelLimitDrawUp', 'SetInfo'): 'economy/PanelLimitDrawUp.SetInfo.lua'}),
    ('PanelLimitDrawUpInvincible.lua', 'scripts/PanelLimitDrawUpInvincible.luac', {('PanelLimitDrawUpInvincible', 'SetInfo'): 'economy/PanelLimitDrawUpInvincible.SetInfo.lua'}),
    ('PanelDrawPaiLianAnimator.lua', 'scripts/PanelDrawPaiLianAnimator.luac', {('PanelDrawPaiLianAnimator', 'Show'): 'economy/PanelDrawPaiLianAnimator.Show.lua'}),
    ('PanelDrawPaiLianAnimatorInvincible.lua', 'scripts/PanelDrawPaiLianAnimatorInvincible.luac', {('PanelDrawPaiLianAnimatorInvincible', 'Show'): 'economy/PanelDrawPaiLianAnimatorInvincible.Show.lua'}),
    ('ResourceHelper.lua', 'scripts/ResourceHelper.luac', {('ResourceHelper', 'GetExpSprite'): 'localization/ResourceHelper.GetExpSprite.lua', ('ResourceHelper', 'CheckResVersion'): 'heroes/ResourceHelper.CheckResVersion.lua'}),
    ('BasicComp.lua', 'scripts/BasicComp.luac', {('BasicComp', 'GetRandomName'): 'localization/BasicComp.GetRandomName.lua'}),
    ('I18NHelper.lua', 'scripts/I18NHelper.luac', {('I18NHelper', 'GetNumFormat'): 'localization/I18NHelper.GetNumFormat.lua'}),
    ('PanelBooks.lua', 'scripts/PanelBooks.luac', {('BookSkinLvGo', 'CalculateSkinLv'): 'gallery/BookSkinLvGo.CalculateSkinLv.lua'}),
    ('ManualComp.lua', 'scripts/ManualComp.luac', {('ManualComp', 'InitBindEasterEggReddot'): 'gallery/ManualComp.InitBindEasterEggReddot.lua'}),
    ('UniMatrixHelper.lua', 'scripts/UniMatrixHelper.luac', {('UniMatrixHelper', n): f'worldchat/UniMatrixHelper.{n}.lua' for n in ('JoinDivisionWorldChannel', 'LeaveChannel', 'SendWorldMessage')}),
    ('NovelMgr.lua', 'scripts/NovelMgr.luac', {('NovelMgr', name): f'novel/NovelMgr.{name}.lua' for name in ('PlayFadeinAnim', 'PlayFadeinEndAnim', 'PlayFadeoutEndAnim')}),
    ('PanelLogin.lua', 'scripts/PanelLogin.luac', {('PanelLogin', name): f'login/PanelLogin.{name}.lua' for name in ('OnLoad', 'DoLogin', 'OnResponsePharosSpeedup', 'RefreshState', 'UpdateVerticalLayout', 'OnUnload', 'UseDebugLogin', '_SwitchForceDebugLogin', 'StartLogin', 'OnClickFix')}),
    ('ServerClient.lua', 'scripts/ServerClient.luac', {('ServerClient', 'CallServerMethod'): 'login/ServerClient.CallServerMethod.lua'}),
    ('PatchUtils.lua', 'scripts/PatchUtils.luac', {('PatchUtils', 'ResetPatch'): 'repair/PatchUtils.ResetPatch.lua', ('PatchUtils', 'RestartOrQuitGame'): 'repair/PatchUtils.RestartOrQuitGame.lua'}),
    ('Status.lua', 'scripts/Status.luac', {('Status', '_OnFixClicked'): 'repair/Status._OnFixClicked.lua'}),
]
FALSE_READS = [
    ('Battle.lua', 'scripts/Battle.luac', 'Battle', 'isPvp', {'BattleEnd': 2, 'SendBattleResult': 1}),
]
STRINGS = 'error_content_cn.lua'
STRINGS_PATH = 'localization/error_content_cn.luac'
DATA_TEXT = 'localization/data_text.json'
CJK = re.compile('[\u3400-\u9fff]')
TRANSLATED = {'draw_pool_data', 'activity_data', 'act_condition_task_data', 'activity_follow_data', 'battlepass_task_data', 'activity_version_preview_data', 'hero_purchase_pack_data', 'gift_data', 'server_list_data', 'draw_prize_data', 'card_data_new_cn', 'character_data_new_cn', 'error_content_cn', 'item_data_cn', 'mail_data_cn', 'shop_data_cn'}
LOCALIZATION = 'localization'
SCRIPTS = 'scripts'


def values(value):
    if isinstance(value, dict):
        for child in value.values():
            yield from values(child)
    elif isinstance(value, list):
        for child in value:
            yield from values(child)
    elif isinstance(value, str):
        yield value


def sha(data):
    return hashlib.sha256(data).hexdigest()


def parse(raw):
    position = 33

    def take(size):
        nonlocal position
        value = raw[position:position + size]
        position += size
        assert len(value) == size
        return value

    def number():
        return struct.unpack('<I', take(4))[0]

    def string():
        size = take(1)[0]
        if size == 255:
            size = number()
        return take(size - 1) if size else None

    def prototype():
        node = {'source': string(), 'header': take(11)}
        node['code'] = [number() for _ in range(number())]
        node['constants'] = []
        for _ in range(number()):
            tag = take(1)[0]
            value = None
            if tag == 1:
                value = take(1)
            elif tag in (3, 19):
                value = take(8)
            elif tag in (4, 20):
                value = string()
            else:
                assert tag == 0, tag
            node['constants'].append((tag, value))
        node['upvalues'] = [take(2) for _ in range(number())]
        node['children'] = [prototype() for _ in range(number())]
        node['lines'] = [number() for _ in range(number())]
        node['locals'] = [(string(), take(8)) for _ in range(number())]
        node['upnames'] = [string() for _ in range(number())]
        return node

    root = prototype()
    assert position == len(raw), (position, len(raw))
    return raw[:33], root


def serialize(header, root):
    data = bytearray(header)

    def number(value):
        data.extend(struct.pack('<I', value))

    def string(value):
        if value is None:
            data.append(0)
            return
        size = len(value) + 1
        if size < 255:
            data.append(size)
        else:
            data.append(255)
            number(size)
        data.extend(value)

    def sequence(values, write):
        number(len(values))
        for value in values:
            write(value)

    def constant(value):
        tag, raw = value
        data.append(tag)
        if tag in (4, 20):
            string(raw)
        elif raw is not None:
            data.extend(raw)

    def prototype(node):
        string(node['source'])
        data.extend(node['header'])
        sequence(node['code'], number)
        sequence(node['constants'], constant)
        sequence(node['upvalues'], data.extend)
        sequence(node['children'], prototype)
        sequence(node['lines'], number)
        sequence(node['locals'], lambda row: (string(row[0]), data.extend(row[1])))
        sequence(node['upnames'], string)

    prototype(root)
    return bytes(data)


def rebuild_constants(original, replacements):
    header, root = parse(original)
    assert serialize(header, root) == original
    count = 0

    def replace(node):
        nonlocal count
        for position, (tag, value) in enumerate(node['constants']):
            if tag not in (4, 20) or value is None:
                continue
            source = value.decode('utf8', errors='surrogateescape')
            if source in replacements:
                encoded = replacements[source].encode('utf8')
                node['constants'][position] = (20 if len(encoded) > 40 else 4, encoded)
                count += 1
        for child in node['children']:
            replace(child)

    replace(root)
    return serialize(header, root), count


def active_locals(root, pc):
    registers, count = {}, 0
    for name, span in root['locals']:
        start, end = struct.unpack('<II', span)
        if start <= pc < end:
            registers[name.decode()] = count
            count += 1
    return registers


class Toolchain:
    def __init__(self, path):
        self.path = path
        sys.path.insert(0, str(path))
        sys.path.insert(0, str(path / 'tools/unitypy'))
        from compile_compatible_lua import compile_source, CUSTOM
        import UnityPy
        UnityPy.config.FALLBACK_UNITY_VERSION = '2021.3.41f1'
        self.compile_source, self.opcodes, self.unity = compile_source, CUSTOM, UnityPy
        lib = C.CDLL(str(path / 'xlua.offline.dll'))
        lib.luaL_newstate.restype = C.c_void_p
        for name in ['luaL_openlibs', 'luaopen_mclua', 'lua_close', 'lua_pushnil']:
            getattr(lib, name).argtypes = [C.c_void_p]
        lib.lua_gettop.argtypes = [C.c_void_p]
        lib.lua_gettop.restype = C.c_int
        lib.lua_settop.argtypes = [C.c_void_p, C.c_int]
        lib.xluaL_loadbuffer.argtypes = [C.c_void_p, C.c_void_p, C.c_int, C.c_char_p]
        lib.xluaL_loadbuffer.restype = C.c_int
        lib.lua_pcall.argtypes = [C.c_void_p, C.c_int, C.c_int, C.c_int]
        lib.lua_pcall.restype = C.c_int
        lib.lua_getfield.argtypes = [C.c_void_p, C.c_int, C.c_char_p]
        lib.lua_type.argtypes = [C.c_void_p, C.c_int]
        lib.lua_type.restype = C.c_int
        lib.lua_next.argtypes = [C.c_void_p, C.c_int]
        lib.lua_next.restype = C.c_int
        lib.lua_tolstring.argtypes = [C.c_void_p, C.c_int, C.POINTER(C.c_size_t)]
        lib.lua_tolstring.restype = C.c_void_p
        lib.lua_tonumberx.argtypes = [C.c_void_p, C.c_int, C.c_void_p]
        lib.lua_tonumberx.restype = C.c_double
        lib.lua_toboolean.argtypes = [C.c_void_p, C.c_int]
        lib.lua_toboolean.restype = C.c_int
        self.lib = lib
        self.state = lib.luaL_newstate()
        lib.luaL_openlibs(self.state)
        lib.luaopen_mclua(self.state)
        lib.lua_settop(self.state, 0)

    def text(self, index):
        size = C.c_size_t()
        pointer = self.lib.lua_tolstring(self.state, index, C.byref(size))
        return C.string_at(pointer, size.value).decode('utf8')

    def check(self, status):
        if status:
            raise RuntimeError(self.text(-1))

    def value(self, index):
        lib, state = self.lib, self.state
        kind = lib.lua_type(state, index)
        if kind == 0:
            return None
        if kind == 1:
            return bool(lib.lua_toboolean(state, index))
        if kind == 3:
            number = lib.lua_tonumberx(state, index, None)
            return int(number) if number.is_integer() else number
        if kind == 4:
            return self.text(index)
        if kind != 5:
            raise ValueError(kind)
        absolute = lib.lua_gettop(state) + index + 1 if index < 0 else index
        result = {}
        lib.lua_pushnil(state)
        while lib.lua_next(state, absolute):
            key = self.value(-2)
            if isinstance(key, dict):
                key = json.dumps(key, sort_keys=True, default=str)
            result[key] = self.value(-1)
            lib.lua_settop(state, -2)
        return result

    def load(self, data, name):
        self.lib.lua_settop(self.state, 0)
        buffer = C.create_string_buffer(data)
        self.check(self.lib.xluaL_loadbuffer(self.state, buffer, len(data), name.encode()))
        self.lib.lua_settop(self.state, 0)

    def table_data(self, data, name):
        lib, state = self.lib, self.state
        lib.lua_settop(state, 0)
        buffer = C.create_string_buffer(data)
        self.check(lib.xluaL_loadbuffer(state, buffer, len(data), name.encode()))
        self.check(lib.lua_pcall(state, 0, 1, 0))
        lib.lua_getfield(state, -1, b'Instance')
        self.check(lib.lua_pcall(state, 0, 1, 0))
        lib.lua_getfield(state, -1, b'data')
        result = self.value(-1)
        lib.lua_settop(state, 0)
        return result

    def module_data(self, data, name):
        lib, state = self.lib, self.state
        lib.lua_settop(state, 0)
        buffer = C.create_string_buffer(data)
        self.check(lib.xluaL_loadbuffer(state, buffer, len(data), name.encode()))
        self.check(lib.lua_pcall(state, 0, 1, 0))
        lib.lua_getfield(state, -1, b'Instance')
        if lib.lua_type(state, -1) == 6:
            self.check(lib.lua_pcall(state, 0, 1, 0))
            lib.lua_getfield(state, -1, b'data')
        else:
            lib.lua_settop(state, -2)
        result = self.value(-1)
        lib.lua_settop(state, 0)
        return result

    def methods(self, root):
        closures, loads, result = {}, {}, {}
        for pc, instruction in enumerate(root['code']):
            op = self.opcodes[instruction & 63]
            register = instruction >> 6 & 255
            key, value = instruction >> 23 & 511, instruction >> 14 & 511
            if op == 'CLOSURE':
                closures[register] = (instruction >> 14, pc)
                loads.pop(register, None)
            elif op == 'LOADK':
                loads[register] = instruction >> 14
            elif op == 'SETTABLE' and value in closures:
                constant = key - 256 if key >= 256 else loads.pop(key, None)
                if constant is None or root['constants'][constant][0] not in (4, 20):
                    continue
                owners = {index: name for name, index in active_locals(root, pc).items()}
                if register not in owners:
                    continue
                method = (owners[register], root['constants'][constant][1].decode())
                assert method not in result, method
                result[method] = closures[value]
        return result

    def rebuild(self, original, replacements):
        header, root = parse(original)
        assert serialize(header, root) == original
        mapping = self.methods(root)
        environment = {name.decode(): index for index, name in enumerate(root['upnames'])}
        for index, pc in mapping.values():
            registers = active_locals(root, pc)
            child = root['children'][index]
            for upname, descriptor in zip(child['upnames'], child['upvalues']):
                upname = upname.decode()
                assert descriptor == (bytes([1, registers[upname]]) if upname in registers else bytes([0, environment[upname]])), upname
        before = {method: serialize(header, root['children'][index]) for method, (index, _) in mapping.items()}
        for (owner, name), body in replacements.items():
            index, pc = mapping[(owner, name)]
            registers = active_locals(root, pc)
            declarations = '\n'.join('local ' + local for local in registers)
            compiled = self.compile_source((declarations + '\n' + body + '\nreturn ' + owner).encode(), '@' + owner + '.lua')
            _, compiled_root = parse(compiled)
            replacement = compiled_root['children'][self.methods(compiled_root)[(owner, name)][0]]
            previous = root['children'][index]
            known = {upname.decode(): descriptor for upname, descriptor in zip(previous['upnames'], previous['upvalues'])}
            upvalues = []
            for upname in replacement['upnames']:
                upname = upname.decode()
                if upname in registers:
                    descriptor = bytes([1, registers[upname]])
                else:
                    descriptor = bytes([0, environment[upname]])
                assert known.get(upname, descriptor) == descriptor, (owner, name, upname)
                upvalues.append(descriptor)
            replacement['upvalues'] = upvalues
            root['children'][index] = replacement
        after = {method: serialize(header, root['children'][index]) for method, (index, _) in mapping.items()}
        changed = sorted(method for method in mapping if before[method] != after[method])
        assert changed == sorted(replacements), changed
        patched = serialize(header, root)
        untouched = [index for method, (index, _) in mapping.items() if method not in replacements]
        _, check = parse(patched)
        _, base = parse(original)
        assert all(serialize(header, check['children'][index]) == serialize(header, base['children'][index]) for index in untouched)
        assert check['code'] == base['code'] and check['constants'] == base['constants'] and check['upvalues'] == base['upvalues']
        return patched, {'changed_methods': ['%s:%s' % method for method in changed], 'preserved_methods': len(mapping) - len(changed)}

    def close(self):
        self.lib.lua_close(self.state)


def literal(value):
    if value is None:
        return 'nil'
    if isinstance(value, bool):
        return 'true' if value else 'false'
    if isinstance(value, (int, float)):
        return repr(value)
    if isinstance(value, str):
        data = value.encode('utf8')
        return '"' + ''.join(chr(c) if 32 <= c < 127 and c not in (34, 92) else '\\' + str(c).zfill(3) for c in data) + '"'
    if isinstance(value, dict):
        return '{' + ','.join('[' + literal(k) + ']=' + literal(v) for k, v in sorted(value.items(), key=lambda p: (type(p[0]).__name__, p[0]))) + '}'
    raise TypeError(type(value))


def constructor(data):
    return 'function error_content_cn:Ctor()\nself.data=' + literal(data) + '\nend\n'


def decompile(toolchain, data, directory, name):
    source = directory / (name + '.luac')
    output = directory / (name + '.lua')
    standard = bytearray(data[:12] + bytes([4]) + data[12:])
    standard[5] = 0
    source.write_bytes(standard)
    command = ['java', '-cp', str(toolchain / 'tools/unluac-main/classes'), 'unluac.Main', '--opmap', str(toolchain / 'opmap.txt'), '-o', str(output), str(source)]
    result = subprocess.run(command, capture_output=True)
    if result.returncode:
        raise RuntimeError(name + ': ' + result.stderr.decode(errors='replace'))
    return output.read_text(encoding='utf8')


def without_methods(source, methods):
    for owner, name in methods:
        match = re.search(rf'(?m)^function {owner}[.:]{name}\(', source)
        assert match, (owner, name)
        following = re.search(r'(?m)^(function |local |return |[A-Za-z_][\w.]* = |rpc\()', source[match.end():])
        end = match.end() + following.start() if following else len(source)
        source = source[:match.start()] + source[end:]
    return source


def false_reads(tools, original, owner, constant, expected):
    header, root = parse(original)
    assert serialize(header, root) == original
    mapping = tools.methods(root)
    patched_root = parse(original)[1]
    changes = []
    for method, count in expected.items():
        index = mapping[(owner, method)][0]
        child = patched_root['children'][index]
        matches = []
        for pc, instruction in enumerate(child['code']):
            key = instruction >> 14 & 511
            if tools.opcodes[instruction & 63] == 'GETTABLE' and key >= 256 and child['constants'][key - 256][0] in (4, 20) and child['constants'][key - 256][1] == constant.encode():
                matches.append(pc)
        assert len(matches) == count, (method, matches)
        pc = matches[-1]
        register = child['code'][pc] >> 6 & 255
        child['code'][pc] = tools.opcodes.index('LOADBOOL') | register << 6
        changes.append('%s:%s@%d' % (owner, method, pc))
    patched = serialize(header, patched_root)
    _, check = parse(patched)
    targets = {mapping[(owner, method)][0] for method in expected}
    for index, (child, base) in enumerate(zip(check['children'], root['children'])):
        if index in targets:
            assert len(child['code']) == len(base['code']) and child['constants'] == base['constants']
            assert sum(left != right for left, right in zip(child['code'], base['code'])) == 1
        else:
            assert serialize(header, child) == serialize(header, base)
    assert check['code'] == root['code'] and check['constants'] == root['constants']
    return patched, {'forced_false_reads': changes}


def build(arguments):
    repo, analysis = arguments.repo, arguments.analysis
    assets = repo / 'assets'
    tools = Toolchain(arguments.toolchain)
    index = {row['name']: row for row in json.loads((analysis / 'installed/index.json').read_text())}
    report = {'modules': {}}
    outputs = {}
    work = Path(tempfile.mkdtemp(prefix='client-modules-'))
    try:
        reads = {}
        for asset, path, owner, constant, expected in FALSE_READS:
            original = (analysis / 'installed' / index[asset]['file']).read_bytes()
            patched, details = false_reads(tools, original, owner, constant, expected)
            tools.load(patched, asset)
            details.update({'original_sha256': sha(original), 'patched_sha256': sha(patched), 'path_id': index[asset]['id']})
            report['modules'][asset] = details
            outputs[asset] = (path, patched)
            reads[asset] = patched
        for asset, path, replacements in MODULES:
            original = reads.get(asset) or (analysis / 'installed' / index[asset]['file']).read_bytes()
            bodies = {method: (SOURCES / file).read_text(encoding='utf8') for method, file in replacements.items()}
            patched, details = tools.rebuild(original, bodies)
            tools.load(patched, asset)
            before = decompile(arguments.toolchain, original, work, 'original-' + asset)
            after = decompile(arguments.toolchain, patched, work, 'patched-' + asset)
            assert without_methods(before, replacements) == without_methods(after, replacements), asset
            for method in replacements:
                assert re.search(rf'(?m)^function {method[0]}[.:]{method[1]}\(', after), method
                assert not re.search(r'\b(pairs|ipairs)\(', re.search(rf'(?ms)^function {method[0]}[.:]{method[1]}\(.*?^end$', after).group(0)), method
            details.update({'original_sha256': sha(original), 'patched_sha256': sha(patched), 'path_id': index[asset]['id'], 'decompiled_untouched_identical': True})
            if asset in reads:
                details['forced_false_reads'] = report['modules'][asset]['forced_false_reads']
            report['modules'][asset] = details
            outputs[asset] = (path, patched)
        localized = {}
        for module in sorted((assets / LOCALIZATION).glob('*.luac')):
            name = module.stem + '.lua'
            assert name in index, name
            localized[name] = ('%s/%s' % (LOCALIZATION, module.name), module.read_bytes())
        for table, replacements in json.loads((SOURCES / DATA_TEXT).read_text(encoding='utf8')).items():
            name = table + '.lua'
            assert name in index, name
            path = '%s/%s.luac' % (LOCALIZATION, table)
            if table.endswith('_cn'):
                source = localized[name][1]
            else:
                source = (analysis / 'installed' / index[name]['file']).read_bytes()
            patched, changed = rebuild_constants(source, replacements)
            remaining = sorted({value for value in values(tools.module_data(patched, table)) if value in replacements or table in TRANSLATED and CJK.search(value)})
            assert not remaining, (table, remaining)
            report['modules'][name] = {'method': 'constants', 'changed_constants': changed, 'original_sha256': sha(source), 'patched_sha256': sha(patched)}
            localized[name] = (path, patched)
        strings_original = (analysis / 'installed' / index[STRINGS]['file']).read_bytes()
        current = localized[STRINGS][1]
        additions = {}
        for group in json.loads((SOURCES / 'localization/strings.json').read_text(encoding='utf8')).values():
            for key, text in group.items():
                assert key not in additions, key
                additions[key] = text
        data = tools.table_data(current, STRINGS)
        for key in additions:
            data.pop(key, None)
        next_id = max(row['id'] for row in data.values()) + 1
        expected = dict(data)
        for offset, (key, text) in enumerate(additions.items()):
            expected[key] = {'error_content': text, 'id': next_id + offset}
        header, root = parse(strings_original)
        ctor = root['children'][tools.methods(root)[('error_content_cn', 'Ctor')][0]]
        permitted = {'LOADBOOL', 'LOADK', 'LOADKX', 'LOADNIL', 'NEWTABLE', 'RETURN', 'SETLIST', 'SETTABLE', 'EXTRAARG', 'MOVE'}
        assert {tools.opcodes[x & 63] for x in ctor['code']} <= permitted
        strings, details = tools.rebuild(strings_original, {('error_content_cn', 'Ctor'): constructor(expected)})
        assert tools.table_data(strings, STRINGS) == expected
        details.update({'original_sha256': sha(strings_original), 'previous_sha256': sha(current), 'patched_sha256': sha(strings), 'added_keys': {key: expected[key]['id'] for key in additions}, 'existing_rows_unchanged': all(expected[key] == data[key] for key in data)})
        report['modules'][STRINGS] = details
        localized[STRINGS] = (STRINGS_PATH, strings)
        for name, (path, data) in localized.items():
            tools.load(data, name)
            assert name not in outputs, name
            outputs[name] = (path, data)
    finally:
        tools.close()
        shutil.rmtree(work, ignore_errors=True)
    bundle = arguments.scripts.read_bytes()
    offset = bundle.find(b'UnityFS\0', 8)
    environment = tools.unity.load(bundle[offset:])
    before = {obj.path_id: obj.get_raw_data() for obj in environment.objects}
    replaced = []
    for obj in environment.objects:
        if obj.type.name != 'TextAsset':
            continue
        tree = obj.read_typetree()
        name = tree['m_Name']
        if name not in outputs:
            continue
        assert obj.path_id == index[name]['id'], name
        tree['m_Script'] = outputs[name][1].decode('utf-8', errors='surrogateescape')
        obj.set_raw_data(obj.save_typetree(tree))
        replaced.append(obj.path_id)
    assert len(replaced) == len(outputs)
    packed = bundle[:offset] + next(iter(environment.files.values())).save(packer='original')
    verified = tools.unity.load(packed[offset:])
    after = {obj.path_id: obj.get_raw_data() for obj in verified.objects}
    assert set(after) == set(before)
    assert {key for key in after if after[key] != before[key]} <= set(replaced)
    for obj in verified.objects:
        if obj.type.name == 'TextAsset':
            tree = obj.read_typetree()
            if tree['m_Name'] in outputs:
                assert tree['m_Script'].encode('utf-8', errors='surrogateescape') == outputs[tree['m_Name']][1]
    report.update({'source_scripts_sha256': sha(bundle), 'scripts_sha256': sha(packed), 'bundle_objects_changed': len(replaced), 'bundle_objects_preserved': len(after) - len(replaced)})
    if not arguments.check:
        for path, patched in outputs.values():
            (assets / path).write_bytes(patched)
        (assets / SCRIPTS / 'scripts.pck').write_bytes(packed)
    print(json.dumps(report, indent=2, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(description='Build the patched client Lua modules and scripts bundle.')
    parser.add_argument('--repo', type=Path, default=REPO)
    parser.add_argument('--analysis', type=Path, required=True, help='client analysis folder with the installed index and bytecode')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder with unluac and opmap')
    parser.add_argument('--scripts', type=Path, required=True, help='pristine scripts bundle of the client resources to patch')
    parser.add_argument('--check', action='store_true', help='verify without writing assets')
    build(parser.parse_args())


if __name__ == '__main__':
    main()
