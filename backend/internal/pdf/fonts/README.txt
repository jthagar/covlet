Bundled TTFs for PDF output via github.com/signintech/gopdf:

- DejaVuSans.ttf — body text
- DejaVuSans-Bold.ttf — title

Font project: https://dejavu-fonts.github.io/
License: https://dejavu-fonts.github.io/License.html

Packages: dejavu-fonts-ttf on npm (same files as upstream).

Optional environment variables (see backend/internal/pdf/pdf.go):

- COVLET_PDF_FONT — path to a custom TTF for body text
- COVLET_PDF_FONT_BOLD — path to a custom TTF for the title (optional; if only COVLET_PDF_FONT is set, it is used for both)
