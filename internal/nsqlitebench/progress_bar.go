package nsqlitebench

import (
	"github.com/schollz/progressbar/v3"
)

type progressBar struct {
	pb          *progressbar.ProgressBar
	ciMode      bool
	description string
	maxItems    int
}

func NewBar(ciMode bool, description string, maxItems int) *progressBar {
	if ciMode {
		return &progressBar{}
	}

	pb := progressbar.Default(int64(maxItems), description)
	_ = pb.Set(0)

	return &progressBar{
		pb:          pb,
		description: description,
		maxItems:    maxItems,
	}
}

func (p *progressBar) Inc() {
	if p.ciMode {
		return
	}

	_ = p.pb.Add(1)
}

func (p *progressBar) Finish() {
	if p.ciMode {
		return
	}

	_ = p.pb.Finish()
	_ = p.pb.Close()
}
