import argparse
import ctypes as C
import json
import sys
from pathlib import Path

sys.dont_write_bytecode = True

WRITER = C.CFUNCTYPE(C.c_int, C.c_void_p, C.c_void_p, C.c_size_t, C.c_void_p)


def extract(arguments):
    sys.path.insert(0, str(arguments.toolchain / 'tools/unitypy'))
    import UnityPy
    UnityPy.config.FALLBACK_UNITY_VERSION = '2021.3.41f1'
    lib = C.CDLL(str(arguments.toolchain / 'xlua.offline.dll'))
    lib.luaL_newstate.restype = C.c_void_p
    lib.lua_close.argtypes = [C.c_void_p]
    lib.lua_settop.argtypes = [C.c_void_p, C.c_int]
    lib.xluaL_loadbuffer.argtypes = [C.c_void_p, C.c_void_p, C.c_int, C.c_char_p]
    lib.lua_dump.argtypes = [C.c_void_p, WRITER, C.c_void_p, C.c_int]
    chunks = []

    @WRITER
    def writer(state, pointer, size, user):
        chunks.append(C.string_at(pointer, size))
        return 0

    out = arguments.analysis / 'installed'
    out.mkdir(parents=True, exist_ok=True)
    raw = arguments.scripts.read_bytes()
    env = UnityPy.load(raw[raw.find(b'UnityFS\0', 8):])
    state = lib.luaL_newstate()
    index = []
    try:
        for obj in env.objects:
            if obj.type.name != 'TextAsset':
                continue
            tree = obj.read_typetree()
            name = tree['m_Name']
            encrypted = tree['m_Script'].encode('utf-8', errors='surrogateescape')
            buffer = C.create_string_buffer(encrypted)
            if lib.xluaL_loadbuffer(state, buffer, len(encrypted), ('@' + name).encode()):
                raise SystemExit(f'{name}: cannot load')
            chunks.clear()
            lib.lua_dump(state, writer, None, 0)
            dumped = b''.join(chunks)
            path = out / f'{obj.path_id}_{name}'
            path.write_bytes(dumped)
            standard = bytearray(dumped)
            standard[5] = 0
            standard[13:13] = bytes([4])
            path.with_suffix('.luac').write_bytes(bytes(standard))
            index.append({'name': name, 'id': obj.path_id, 'file': path.name, 'size': len(dumped)})
            lib.lua_settop(state, 0)
    finally:
        lib.lua_close(state)
    (out / 'index.json').write_text(json.dumps(index, indent=2))
    print(json.dumps({'modules': len(index)}))


def main():
    parser = argparse.ArgumentParser(description='Extract the Lua modules of a client scripts bundle into an analysis folder (installed/index.json and bytecode).')
    parser.add_argument('--scripts', type=Path, required=True, help='scripts.pck of the client version to analyse')
    parser.add_argument('--analysis', type=Path, required=True, help='analysis folder to create or refresh')
    parser.add_argument('--toolchain', type=Path, required=True, help='offline Lua toolchain folder with xlua.offline.dll and tools/unitypy')
    extract(parser.parse_args())


if __name__ == '__main__':
    main()
