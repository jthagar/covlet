package pdf

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/signintech/gopdf"
)

//go:embed fonts/DejaVuSans.ttf
var dejaVuSansTTF []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var dejaVuSansBoldTTF []byte

const (
	maxPDFInputBytes = 2 << 20 // 2 MiB combined title + body
	maxFontFileBytes = 5 << 20 // 5 MiB per custom TTF
	marginPt         = 50.0
	titleFontSize    = 14.0
	bodyFontSize     = 11.0
	fontFamilyReg    = "covlet"
	fontFamilyBold   = "covlet-bold"
)

// TextToPDF renders title and plain UTF-8 body into a PDF (A4, multiple pages if needed).
// Default fonts: embedded DejaVu Sans and DejaVu Sans Bold (see fonts/README.txt).
//
// Optional env (long-term customization):
//   - COVLET_PDF_FONT: path to a TTF for body text (replaces embedded regular).
//   - COVLET_PDF_FONT_BOLD: path to a TTF for the title (defaults to embedded bold, or the same file as COVLET_PDF_FONT if only that is set).
func TextToPDF(title, body string) ([]byte, error) {
	if len(title)+len(body) > maxPDFInputBytes {
		return nil, fmt.Errorf("content exceeds maximum size (%d bytes)", maxPDFInputBytes)
	}
	if !utf8.ValidString(title) || !utf8.ValidString(body) {
		return nil, fmt.Errorf("invalid UTF-8 in title or body")
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return nil, fmt.Errorf("nothing to render")
	}

	regular, bold, err := loadFontBytes()
	if err != nil {
		return nil, err
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	if err := pdf.AddTTFFontData(fontFamilyReg, regular); err != nil {
		return nil, fmt.Errorf("load body font: %w", err)
	}
	if err := pdf.AddTTFFontData(fontFamilyBold, bold); err != nil {
		return nil, fmt.Errorf("load title font: %w", err)
	}

	pdf.SetMargins(marginPt, marginPt, marginPt, marginPt)
	pdf.SetXY(marginPt, marginPt)

	textW := gopdf.PageSizeA4.W - 2*marginPt

	if title != "" {
		if err := pdf.SetFont(fontFamilyBold, "", titleFontSize); err != nil {
			return nil, err
		}
		lineH, err := pdf.MeasureCellHeightByText("Hg")
		if err != nil {
			return nil, err
		}
		if err := writeWordWrapped(&pdf, marginPt, textW, lineH, title); err != nil {
			return nil, err
		}
		if body != "" {
			pdf.Br(lineH * 0.75)
		}
	}

	if body != "" {
		if err := pdf.SetFont(fontFamilyReg, "", bodyFontSize); err != nil {
			return nil, err
		}
		lineH, err := pdf.MeasureCellHeightByText("Hg")
		if err != nil {
			return nil, err
		}
		if err := writeWordWrapped(&pdf, marginPt, textW, lineH, body); err != nil {
			return nil, err
		}
	}

	if title != "" {
		pdf.SetInfo(gopdf.PdfInfo{
			Title:   title,
			Subject: "covlet export",
			Creator: "covlet",
		})
	} else {
		pdf.SetInfo(gopdf.PdfInfo{
			Subject: "covlet export",
			Creator: "covlet",
		})
	}

	return pdf.GetBytesPdfReturnErr()
}

func loadFontBytes() (regular []byte, bold []byte, err error) {
	customReg := strings.TrimSpace(os.Getenv("COVLET_PDF_FONT"))
	customBold := strings.TrimSpace(os.Getenv("COVLET_PDF_FONT_BOLD"))

	if customReg != "" {
		regular, err = readFontFile(customReg)
		if err != nil {
			return nil, nil, err
		}
	} else {
		regular = dejaVuSansTTF
	}

	switch {
	case customBold != "":
		bold, err = readFontFile(customBold)
		if err != nil {
			return nil, nil, err
		}
	case customReg != "":
		bold = regular
	default:
		bold = dejaVuSansBoldTTF
	}

	return regular, bold, nil
}

func readFontFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read font %q: %w", path, err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("empty font file %q", path)
	}
	if len(b) > maxFontFileBytes {
		return nil, fmt.Errorf("font file %q exceeds maximum size (%d bytes)", path, maxFontFileBytes)
	}
	return b, nil
}

func writeWordWrapped(pdf *gopdf.GoPdf, marginLeft, textW, lineH float64, text string) error {
	lines, err := pdf.SplitTextWithWordWrap(text, textW)
	if err != nil {
		if errors.Is(err, gopdf.ErrEmptyString) {
			return nil
		}
		return err
	}
	for _, line := range lines {
		pdf.SetNewY(pdf.GetY(), lineH)
		pdf.SetX(marginLeft)
		if err := pdf.Cell(&gopdf.Rect{W: textW, H: lineH}, line); err != nil {
			return err
		}
		pdf.Br(lineH)
	}
	return nil
}
