// Package fixture generates a deterministic "dirty project repository" for
// tests and demos: oversized files of several kinds, an archive, an
// installer, junk, an empty folder, a zero-size file, exact duplicates and
// copy-named near-duplicates.
package fixture

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
)

const (
	kb = 1 << 10
	mb = 1 << 20
)

// Options tunes generation.
type Options struct {
	// Scale divides the sizes of the big files (1 = real sizes: a 101 MB
	// video, 16 MB office files…). Tests use 1024 together with thresholds
	// divided by the same factor, so every "normal" file below must stay
	// under 4 KB (images) / 15 KB (documents) / 100 KB (anything) and
	// "Проект A" must remain the heaviest folder at both scales.
	Scale int64
}

// File describes one generated file.
type File struct {
	Rel  string // slash-separated path inside the fixture root
	Size int64  // bytes at Scale 1
	Big  bool   // scaled by Options.Scale
	Seed string // content seed; equal seeds + sizes give identical content
}

// EmptyDirs are created without any content.
var EmptyDirs = []string{"Проект B/Пустая", "Проект B/Пустая вложенная/уровень 2"}

// Files is the fixture layout. Paths are Russian on purpose — that is what
// the tool will meet in real repositories.
var Files = []File{
	// normal documents
	{Rel: "Проект A/Договоры/Договор.docx", Size: 4 * kb, Seed: "contract"},
	{Rel: "Проект A/Договоры/Спецификация.xlsx", Size: 6 * kb, Seed: "spec"},
	{Rel: "Проект A/Отчёты/Отчёт за март.docx", Size: 5 * kb, Seed: "report-march"},
	{Rel: "Проект B/Фото/IMG_0002.jpg", Size: 4 * kb, Seed: "img2"},
	{Rel: "Проект B/Протоколы/Протокол 2026-02-10.pdf", Size: 8 * kb, Seed: "minutes"},
	// oversized by kind (Big → scaled)
	{Rel: "Проект A/Записи/Встреча 2026-03-01.mp4", Size: 101 * mb, Big: true, Seed: "video"},
	{Rel: "Проект A/Презентации/Презентация для заказчика.pptx", Size: 16 * mb, Big: true, Seed: "deck"},
	{Rel: "Проект A/Отчёты/Итоговый отчёт.docx", Size: 16 * mb, Big: true, Seed: "final"},
	{Rel: "Проект B/Фото/IMG_0001.jpg", Size: 6 * mb, Big: true, Seed: "img1"},
	{Rel: "Проект B/Протоколы/Сканы протоколов.pdf", Size: 31 * mb, Big: true, Seed: "scans"},
	// archive and distributive
	{Rel: "Проект B/Старое/Архив проекта 2024.zip", Size: 16 * kb, Seed: "zip"},
	{Rel: "Проект B/Дистрибутивы/setup.exe", Size: 16 * kb, Seed: "exe"},
	// junk
	{Rel: "Проект A/Thumbs.db", Size: 2 * kb, Seed: "thumbs"},
	{Rel: "Проект A/Отчёты/~$Отчёт за март.docx", Size: 162, Seed: "lock"},
	{Rel: "Проект B/.DS_Store", Size: 6 * kb, Seed: "dsstore"},
	{Rel: "Проект B/Старое/Смета.bak", Size: 3 * kb, Seed: "bak"},
	// zero-size file
	{Rel: "Проект A/пустой документ.txt", Size: 0, Seed: "zero"},
	// exact duplicates of the contract (same seed, same size)
	{Rel: "Проект B/Копия Договор.docx", Size: 4 * kb, Seed: "contract"},
	{Rel: "Проект B/Старое/Договор - копия (2).docx", Size: 4 * kb, Seed: "contract"},
	// near-duplicate by name, different content
	{Rel: "Проект A/Отчёты/Отчёт за март (1).docx", Size: 5 * kb, Seed: "report-march-v2"},
}

// Generate writes the fixture into dir, which must be empty or absent.
func Generate(dir string, opt Options) error {
	if opt.Scale <= 0 {
		opt.Scale = 1
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return errors.New(dir + ": folder is not empty")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, d := range EmptyDirs {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(d)), 0o755); err != nil {
			return err
		}
	}
	for _, f := range Files {
		size := f.Size
		if f.Big {
			size /= opt.Scale
		}
		p := filepath.Join(dir, filepath.FromSlash(f.Rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := writeFile(p, size, f.Seed); err != nil {
			return fmt.Errorf("%s: %w", f.Rel, err)
		}
	}
	return nil
}

// writeFile fills the file with pseudo-random bytes derived from seed, so
// equal seeds produce byte-identical files of equal size.
func writeFile(path string, size int64, seed string) error {
	h := fnv.New64a()
	io.WriteString(h, seed)
	rng := rand.New(rand.NewPCG(h.Sum64(), 0x5eed))

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 1*mb)
	for size > 0 {
		n := int64(len(buf))
		if size < n {
			n = size
		}
		chunk := buf[:n]
		for i := 0; i+8 <= len(chunk); i += 8 {
			v := rng.Uint64()
			for j := range 8 {
				chunk[i+j] = byte(v >> (8 * j))
			}
		}
		for i := len(chunk) &^ 7; i < len(chunk); i++ {
			chunk[i] = byte(rng.Uint32())
		}
		if _, err := f.Write(chunk); err != nil {
			return err
		}
		size -= n
	}
	return f.Close()
}
