package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtVersion   = 16
	iconGroupID = 1
	langEnUS    = 0x0409
)

type windowsVersionStrings struct {
	Language    uint16
	Name        string
	FileName    string
	Company     string
	Copyright   string
	Version     string
	BuildNumber int
}

func iconResource(images [][]byte, sizes []int, goarch string) ([]byte, error) {
	return windowsResource(images, sizes, goarch, nil)
}

type resourceLanguage struct {
	id   uint32
	data []byte
}
type resourceID struct {
	id    uint32
	langs []resourceLanguage
}
type resourceType struct {
	id    uint32
	items []resourceID
}

// windowsResource writes the icon and VERSIONINFO in one .rsrc section. The
// Go linker rejects a second .rsrc section, so the two resources must share the
// same COFF object.
func windowsResource(images [][]byte, sizes []int, goarch string, versions []windowsVersionStrings) ([]byte, error) {
	machine, reloc := uint16(0), uint16(0)
	switch goarch {
	case "amd64":
		machine, reloc = 0x8664, 0x0003
	case "arm64":
		machine, reloc = 0xaa64, 0x0002
	case "386":
		machine, reloc = 0x014c, 0x0007
	default:
		return nil, fmt.Errorf("no Windows resources for %s", goarch)
	}
	put := func(b *bytes.Buffer, v any) { _ = binary.Write(b, binary.LittleEndian, v) }
	var group bytes.Buffer
	put(&group, [3]uint16{0, 1, uint16(len(images))})
	for i, img := range images {
		side := uint8(sizes[i])
		if sizes[i] >= 256 {
			side = 0
		}
		put(&group, [4]uint8{side, side, 0, 0})
		put(&group, [2]uint16{1, 32})
		put(&group, uint32(len(img)))
		put(&group, uint16(i+1))
	}
	types := []resourceType{}
	if len(images) > 0 {
		icons := resourceType{id: rtIcon}
		for i, img := range images {
			icons.items = append(icons.items, resourceID{id: uint32(i + 1), langs: []resourceLanguage{{langEnUS, img}}})
		}
		types = append(types, icons, resourceType{id: rtGroupIcon, items: []resourceID{{id: iconGroupID, langs: []resourceLanguage{{langEnUS, group.Bytes()}}}}})
	}
	if len(versions) > 0 {
		version := resourceType{id: rtVersion}
		item := resourceID{id: 1}
		for _, v := range versions {
			item.langs = append(item.langs, resourceLanguage{uint32(v.Language), versionInfo(v)})
		}
		version.items = append(version.items, item)
		types = append(types, version)
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("Windows resource has neither an icon nor version information")
	}
	const dirSize, entrySize, dataEntrySize = 16, 8, 16
	leaves := 0
	for _, t := range types {
		for _, item := range t.items {
			leaves += len(item.langs)
		}
	}
	idDirs := dirSize + entrySize*len(types)
	langDirs := idDirs
	for _, t := range types {
		langDirs += dirSize + entrySize*len(t.items)
	}
	dataEntries := langDirs
	for _, t := range types {
		for _, item := range t.items {
			dataEntries += dirSize + entrySize*len(item.langs)
		}
	}
	data := dataEntries + leaves*dataEntrySize
	var sec bytes.Buffer
	dir := func(entries int) { put(&sec, [4]uint32{0, 0, 0, uint32(entries) << 16}) }
	const subdir = 0x80000000
	dir(len(types))
	at := idDirs
	for _, t := range types {
		put(&sec, [2]uint32{t.id, subdir | uint32(at)})
		at += dirSize + entrySize*len(t.items)
	}
	at = langDirs
	for _, t := range types {
		dir(len(t.items))
		for _, item := range t.items {
			put(&sec, [2]uint32{item.id, subdir | uint32(at)})
			at += dirSize + entrySize*len(item.langs)
		}
	}
	at = dataEntries
	for _, t := range types {
		for _, item := range t.items {
			dir(len(item.langs))
			for _, lang := range item.langs {
				put(&sec, [2]uint32{lang.id, uint32(at)})
				at += dataEntrySize
			}
		}
	}
	var relocs []uint32
	at = data
	for _, t := range types {
		for _, item := range t.items {
			for _, lang := range item.langs {
				relocs = append(relocs, uint32(sec.Len()))
				put(&sec, [4]uint32{uint32(at), uint32(len(lang.data)), 0, 0})
				at += align8(len(lang.data))
			}
		}
	}
	for _, t := range types {
		for _, item := range t.items {
			for _, lang := range item.langs {
				sec.Write(lang.data)
				sec.Write(make([]byte, align8(len(lang.data))-len(lang.data)))
			}
		}
	}
	const fileHeader, sectionHeader, relocSize = 20, 40, 10
	rawData := fileHeader + sectionHeader
	relocations := rawData + sec.Len()
	symbols := relocations + relocSize*len(relocs)
	var out bytes.Buffer
	put(&out, machine)
	put(&out, uint16(1))
	put(&out, [3]uint32{0, uint32(symbols), 1})
	put(&out, [2]uint16{0, 0})
	out.WriteString(".rsrc\x00\x00\x00")
	put(&out, [4]uint32{0, 0, uint32(sec.Len()), uint32(rawData)})
	put(&out, [2]uint32{uint32(relocations), 0})
	put(&out, [2]uint16{uint16(len(relocs)), 0})
	put(&out, uint32(0x40000040))
	out.Write(sec.Bytes())
	for _, r := range relocs {
		put(&out, [2]uint32{r, 0})
		put(&out, reloc)
	}
	out.WriteString(".rsrc\x00\x00\x00")
	put(&out, uint32(0))
	put(&out, int16(1))
	put(&out, uint16(0))
	put(&out, [2]uint8{3, 0})
	put(&out, uint32(4))
	return out.Bytes(), nil
}

func versionInfo(v windowsVersionStrings) []byte {
	parts := versionParts(v.Version)
	if v.BuildNumber > 0 {
		parts[3] = uint16(v.BuildNumber)
	}
	var fixed bytes.Buffer
	put := func(x uint32) { _ = binary.Write(&fixed, binary.LittleEndian, x) }
	put(0xFEEF04BD)
	put(0x00010000)
	put(uint32(parts[0])<<16 | uint32(parts[1]))
	put(uint32(parts[2])<<16 | uint32(parts[3]))
	put(uint32(parts[0])<<16 | uint32(parts[1]))
	put(uint32(parts[2])<<16 | uint32(parts[3]))
	put(0)
	put(0)
	put(0x00040004)
	put(1)
	put(0)
	put(0)
	put(0)
	var table bytes.Buffer
	for _, pair := range [][2]string{
		{"CompanyName", v.Company}, {"FileDescription", v.Name}, {"FileVersion", v.Version},
		{"InternalName", v.FileName + ".exe"}, {"LegalCopyright", v.Copyright}, {"OriginalFilename", v.FileName + ".exe"},
		{"ProductName", v.Name}, {"ProductVersion", v.Version},
	} {
		if pair[1] != "" {
			table.Write(versionBlock(1, pair[0], utf16Bytes(pair[1])))
		}
	}
	tableKey := fmt.Sprintf("%04X%04X", v.Language, 1200)
	stringsBlock := versionBlock(1, "StringFileInfo", versionBlock(1, tableKey, table.Bytes()))
	var translation bytes.Buffer
	_ = binary.Write(&translation, binary.LittleEndian, v.Language)
	_ = binary.Write(&translation, binary.LittleEndian, uint16(1200))
	varBlock := versionBlock(0, "Translation", translation.Bytes())
	varFile := versionBlock(1, "VarFileInfo", varBlock)
	value := append(fixed.Bytes(), stringsBlock...)
	value = append(value, varFile...)
	return versionBlock(0, "VS_VERSION_INFO", value)
}

func versionBlock(typ uint16, key string, value []byte) []byte {
	header := 6 + len(utf16Bytes(key))
	valueStart := align4(header)
	valueLength := 0
	switch {
	case key == "VS_VERSION_INFO":
		valueLength = 52
	case key == "Translation":
		valueLength = len(value)
	case typ == 1 && !isVersionContainer(key):
		valueLength = len(value) / 2
	}
	valueBytes := len(value)
	length := align4(valueStart + valueBytes)
	b := bytes.NewBuffer(nil)
	_ = binary.Write(b, binary.LittleEndian, uint16(length))
	_ = binary.Write(b, binary.LittleEndian, uint16(valueLength))
	_ = binary.Write(b, binary.LittleEndian, typ)
	b.Write(utf16Bytes(key))
	b.Write(make([]byte, valueStart-b.Len()))
	b.Write(value)
	b.Write(make([]byte, length-b.Len()))
	return b.Bytes()
}

func isVersionContainer(key string) bool {
	if key == "StringFileInfo" || key == "VarFileInfo" {
		return true
	}
	if len(key) != 8 {
		return false
	}
	for _, r := range key {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func utf16Bytes(s string) []byte {
	u := append(utf16.Encode([]rune(s)), 0)
	b := make([]byte, len(u)*2)
	for i, x := range u {
		binary.LittleEndian.PutUint16(b[i*2:], x)
	}
	return b
}

func versionParts(v string) [4]uint16 {
	var out [4]uint16
	for i, part := range strings.Split(strings.TrimPrefix(v, "v"), ".") {
		if i == 4 {
			break
		}
		n, _ := strconv.Atoi(part)
		out[i] = uint16(n)
	}
	return out
}

// windowsVersionStringsFor reads the app's catalog languages and maps them to
// Windows language IDs. A catalog without appName still gets a table using the
// project name, so VERSIONINFO is present for every catalog language.
func windowsVersionStringsFor(p *project) ([]windowsVersionStrings, error) {
	company := p.Windows.Publisher
	if company == "" {
		company = p.Windows.Store.PublisherName
	}
	names, err := p.appNames()
	if err != nil {
		return nil, err
	}
	languages := []string{p.I18n.Source}
	if p.I18n.Dir != "" && hasCatalogs(p.I18n.Dir) {
		c, ds := loadCatalog(p.I18n.Dir, p.I18n.Source)
		if err := ds.err(); err != nil {
			return nil, err
		}
		languages = c.languages()
	}
	out := make([]windowsVersionStrings, 0, len(languages))
	seen := map[uint16]bool{}
	for _, lang := range languages {
		id := windowsLanguageID(lang)
		if seen[id] {
			continue
		}
		seen[id] = true
		name := p.Name
		if names != nil && names.names[lang] != "" {
			name = names.names[lang]
		}
		out = append(out, windowsVersionStrings{Language: id, Name: name, FileName: p.Name, Company: company, Copyright: p.Copyright, Version: p.Version, BuildNumber: p.BuildNumber})
	}
	return out, nil
}

func windowsLanguageID(lang string) uint16 {
	ids := map[string]uint16{
		"en": 0x0409, "en-GB": 0x0809, "de": 0x0407, "fr": 0x040c, "es": 0x0c0a,
		"it": 0x0410, "pt": 0x0816, "pt-BR": 0x0416, "ja": 0x0411, "ko": 0x0412,
		"ru": 0x0419, "uk": 0x0422, "pl": 0x0415, "tr": 0x041f, "nl": 0x0413,
		"sv": 0x041d, "da": 0x0406, "nb": 0x0414, "fi": 0x040b, "cs": 0x0405,
		"ar": 0x0401, "bg": 0x0402, "ca": 0x0403, "el": 0x0408, "he": 0x040d,
		"hu": 0x040e, "id": 0x0421, "lt": 0x0427, "ro": 0x0418, "sk": 0x041b,
		"sl": 0x0424, "sr": 0x0c1a, "th": 0x041e, "vi": 0x042a, "eo": 0x046b,
		"zh-Hans": 0x0804, "zh-Hant": 0x0404, "zh-CN": 0x0804, "zh-TW": 0x0404,
	}
	if id, ok := ids[lang]; ok {
		return id
	}
	base, _, _ := strings.Cut(lang, "-")
	if id, ok := ids[base]; ok {
		return id
	}
	return langEnUS
}

func align4(n int) int { return (n + 3) &^ 3 }
func align8(n int) int { return (n + 7) &^ 7 }
