package executable_worker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/photoview/photoview/api/log"
	"github.com/photoview/photoview/api/scanner/scanner_compressfile"
)

type VipsEncoder struct {
	initialized bool
}

func newVipsEncoder() *VipsEncoder {
	vips.Startup(nil)

	log.Info(nil, "Found vips worker", "version", vips.Version)

	return &VipsEncoder{initialized: true}
}

func (cli *VipsEncoder) Terminate() {
	cli.initialized = false
	vips.Shutdown()
}

func (cli *VipsEncoder) IsInstalled() bool {
	return cli != nil && cli.initialized
}

func (cli *VipsEncoder) Encode(inputPath string, outputPath string, quality uint) error {
	image, err := cli.loadImageFromFile(inputPath)
	if err != nil {
		return err
	}
	defer image.Close()

	image.AutoRotate()

	return cli.exportImage(image, outputPath, int(quality))
}

func (cli *VipsEncoder) GenerateThumbnail(inputPath string, outputPath string, width, height uint) error {
	image, err := cli.loadImageFromFile(inputPath)
	if err != nil {
		return err
	}
	defer image.Close()

	image.ThumbnailWithSize(int(width), int(height), vips.InterestingNone, vips.SizeBoth)

	return cli.exportImage(image, outputPath, 70)
}

func (cli *VipsEncoder) IdentifyDimension(inputPath string) (width, height uint, reterr error) {
	image, err := cli.loadImageFromFile(inputPath)
	if err != nil {
		return 0, 0, err
	}
	defer image.Close()

	return uint(image.Width()), uint(image.Height()), nil
}

func (cli *VipsEncoder) loadImageFromFile(inputPath string) (*vips.ImageRef, error) {
	if scanner_compressfile.IsArchiveFilePath(inputPath) {
		data, err := scanner_compressfile.Loader.LoadFile(inputPath)
		if err != nil {
			return nil, err
		}

		return vips.NewImageFromBuffer(data)
	} else {
		return vips.NewImageFromFile(inputPath)
	}
}

func (cli *VipsEncoder) exportImage(image *vips.ImageRef, outputPath string, quality int) error {
	ext := filepath.Ext(outputPath)
	var data []byte
	var err error
	switch ext {
	case ".jpg", ".jpeg":
		p := vips.NewJpegExportParams()
		p.Quality = quality
		data, _, err = image.ExportJpeg(p)
	case ".png":
		p := vips.NewPngExportParams()
		p.Quality = quality
		data, _, err = image.ExportPng(p)
	case ".avif":
		p := vips.NewAvifExportParams()
		p.Quality = quality
		data, _, err = image.ExportAvif(p)
	case ".webp":
		p := vips.NewWebpExportParams()
		p.Quality = quality
		data, _, err = image.ExportWebp(p)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("export image %q error: %w", outputPath, err)
	}

	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("write image %q error: %w", outputPath, err)
	}

	return nil
}
