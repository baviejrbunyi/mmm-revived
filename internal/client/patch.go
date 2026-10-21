package client

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	sessionKeyRVA      uint32 = 0x1413f0
	verifierRVA        uint32 = 0xc71f0
	moduleVerifierSize uint32 = 49
)

var (
	sessionKeyPrologue = []byte{0x40, 0x55, 0x53, 0x56, 0x57}
	verifierPrologue   = []byte{0x48, 0x89, 0x5c, 0x24, 0x18}
)

func PatchClientModules(original, publicKey []byte, modules [][]byte) ([]byte, error) {
	profile, err := profileForDLL(original)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(modules))
	approved := make(map[string]bool)
	for _, file := range profile.Modules() {
		approved[file.SHA256] = true
	}
	for _, module := range modules {
		digest := hash(module)
		if !approved[digest] || seen[digest] {
			return nil, errors.New("client module does not match an approved patch for this build")
		}
		seen[digest] = true
	}
	f, err := pe.NewFile(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	header, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return nil, errors.New("client DLL is not x64")
	}
	if len(publicKey) < 64 || len(publicKey) > 8192 || !bytes.Contains(publicKey, []byte("BEGIN RSA PUBLIC KEY")) {
		return nil, errors.New("invalid public key format")
	}
	peOffset := binary.LittleEndian.Uint32(original[0x3c:])
	sectionTable := peOffset + 24 + uint32(f.SizeOfOptionalHeader)
	newHeader := sectionTable + uint32(f.NumberOfSections)*40
	if newHeader+40 > header.SizeOfHeaders {
		return nil, errors.New("no free PE section header")
	}
	patchOffset, verifierOffset := fileOffset(f.Sections, sessionKeyRVA), fileOffset(f.Sections, verifierRVA)
	if patchOffset == 0 || !bytes.Equal(original[patchOffset:patchOffset+5], sessionKeyPrologue) {
		return nil, errors.New("session key prologue mismatch")
	}
	last := f.Sections[len(f.Sections)-1]
	rva := align(last.VirtualAddress+max(last.VirtualSize, last.Size), header.SectionAlignment)
	raw := align(uint32(len(original)), header.FileAlignment)
	stub := []byte{0x48, 0x8d, 0x05, 0, 0, 0, 0, 0x48, 0x89, 0x44, 0x24, 0x48, 0x48, 0xc7, 0x44, 0x24, 0x50, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(stub[17:], uint32(len(publicKey)))
	stub = append(stub, sessionKeyPrologue...)
	jumpOffset := uint32(len(stub))
	stub = append(stub, 0xe9, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(stub[jumpOffset+1:], uint32(int64(sessionKeyRVA+5)-int64(rva+jumpOffset+5)))
	keyOffset := align(uint32(len(stub)), 8)
	binary.LittleEndian.PutUint32(stub[3:], keyOffset-7)
	size := keyOffset + uint32(len(publicKey)) + 1
	var verifyCode []byte
	var verifyOffset uint32
	moduleOffsets := make([]uint32, len(modules))
	if len(modules) > 0 {
		if verifierOffset == 0 || !bytes.Equal(original[verifierOffset:verifierOffset+5], verifierPrologue) {
			return nil, errors.New("client module verifier prologue mismatch")
		}
		verifyOffset = align(size, 16)
		trampoline := verifyOffset + uint32(len(modules))*moduleVerifierSize
		size = align(trampoline+10, 16)
		verifyCode = make([]byte, 0, trampoline+10-verifyOffset)
		for index, module := range modules {
			moduleOffsets[index] = size
			size = align(size+uint32(len(module)), 16)
			entry := rva + verifyOffset + uint32(index)*moduleVerifierSize
			fallback := rva + trampoline
			if index+1 < len(modules) {
				fallback = entry + moduleVerifierSize
			}
			verifyCode = append(verifyCode, moduleVerifier(entry, rva+moduleOffsets[index], uint32(len(module)), fallback)...)
		}
		verifyCode = append(verifyCode, verifierPrologue...)
		verifyCode = append(verifyCode, 0xe9, 0, 0, 0, 0)
		binary.LittleEndian.PutUint32(verifyCode[len(verifyCode)-4:], uint32(int64(verifierRVA+5)-int64(rva+trampoline+10)))
	}
	rawSize := align(size, header.FileAlignment)
	result := make([]byte, raw+rawSize)
	copy(result, original)
	copy(result[raw:], stub)
	copy(result[raw+keyOffset:], publicKey)
	if len(modules) > 0 {
		copy(result[raw+verifyOffset:], verifyCode)
		for index, module := range modules {
			copy(result[raw+moduleOffsets[index]:], module)
		}
		result[verifierOffset] = 0xe9
		binary.LittleEndian.PutUint32(result[verifierOffset+1:], uint32(int64(rva+verifyOffset)-int64(verifierRVA+5)))
	}
	section := result[newHeader : newHeader+40]
	copy(section, ".mmmcfg")
	binary.LittleEndian.PutUint32(section[8:], size)
	binary.LittleEndian.PutUint32(section[12:], rva)
	binary.LittleEndian.PutUint32(section[16:], rawSize)
	binary.LittleEndian.PutUint32(section[20:], raw)
	binary.LittleEndian.PutUint32(section[36:], 0x60000020)
	binary.LittleEndian.PutUint16(result[peOffset+6:], f.NumberOfSections+1)
	optional := peOffset + 24
	binary.LittleEndian.PutUint32(result[optional+4:], header.SizeOfCode+rawSize)
	binary.LittleEndian.PutUint32(result[optional+56:], align(rva+size, header.SectionAlignment))
	binary.LittleEndian.PutUint32(result[optional+64:], 0)
	clear(result[optional+112+4*8 : optional+112+5*8])
	result[patchOffset] = 0xe9
	binary.LittleEndian.PutUint32(result[patchOffset+1:], uint32(int64(rva)-int64(sessionKeyRVA+5)))
	check, err := pe.NewFile(bytes.NewReader(result))
	if err != nil {
		return nil, fmt.Errorf("validate patched PE: %w", err)
	}
	check.Close()
	return result, nil
}

func moduleVerifier(rva, moduleRVA, size, fallbackRVA uint32) []byte {
	code := []byte{0x48, 0x81, 0xfa, 0, 0, 0, 0, 0x75, 0, 0x56, 0x57, 0x49, 0x89, 0xc9, 0x48, 0x89, 0xce, 0x48, 0x8d, 0x3d, 0, 0, 0, 0, 0xb9, 0, 0, 0, 0, 0xf3, 0xa6, 0x5f, 0x5e, 0x4c, 0x89, 0xc9, 0x75, 0, 0xb8, 1, 0, 0, 0, 0xc3}
	binary.LittleEndian.PutUint32(code[3:], size)
	binary.LittleEndian.PutUint32(code[20:], uint32(int64(moduleRVA)-int64(rva+24)))
	binary.LittleEndian.PutUint32(code[25:], size)
	code[8] = byte(len(code) - 9)
	code[37] = byte(len(code) - 38)
	code = append(code, 0xe9, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(code[len(code)-4:], uint32(int64(fallbackRVA)-int64(rva+uint32(len(code)))))
	return code
}

func fileOffset(sections []*pe.Section, rva uint32) uint32 {
	var offset uint32
	for _, section := range sections {
		if rva >= section.VirtualAddress && rva < section.VirtualAddress+section.Size {
			offset = section.Offset + rva - section.VirtualAddress
		}
	}
	return offset
}

func align(value, boundary uint32) uint32 {
	return (value + boundary - 1) / boundary * boundary
}
