/*

BOOSTER-WEB: Web interface to BOOSTER (https://github.com/evolbioinfo/booster)
Alternative method to compute bootstrap branch supports in large trees.

Copyright (C) 2017 BOOSTER-WEB dev team

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

*/

// Package monitoring to send daily monitoring reports
package monitoring

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"image/color"
	"io"
	"sort"
	"time"

	"github.com/evolbioinfo/booster-web/database"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type MonitorInformation struct {
	Plot          template.URL `json:"plot"` // Base64 representation of the SVG image, as a safe template URL
	PendingJobs   int          `json:"pendingjobs"`
	RunningJobs   int          `json:"runningjobs"`
	FinishedJobs  int          `json:"finishedjobs"`
	CanceledJobs  int          `json:"canceledjobs"`
	ErrorJobs     int          `json:"errorjobs"`
	TimeoutJobs   int          `json:"timeoutjobs"`
	AvgJobsPerDay float64      `json:"avgjobsperday"`
}

func Monitor(db database.BoosterwebDB) (m *MonitorInformation, err error) {
	var plot string
	var pendingJobs, runningJobs, finishedJobs, canceledJobs, errorJobs, timeoutJobs int
	var avgJobsPerDay float64

	if plot, err = plotMonitor(db); err != nil {
		return
	}

	if pendingJobs, runningJobs, finishedJobs, canceledJobs, errorJobs, timeoutJobs, avgJobsPerDay, err = db.GetAnalysesStats(); err != nil {
		return
	}

	m = &MonitorInformation{
		Plot:          template.URL(plot),
		PendingJobs:   pendingJobs,
		RunningJobs:   runningJobs,
		FinishedJobs:  finishedJobs,
		CanceledJobs:  canceledJobs,
		ErrorJobs:     errorJobs,
		TimeoutJobs:   timeoutJobs,
		AvgJobsPerDay: avgJobsPerDay,
	}
	return
}

// Generates and returns a Plot showing the evolution of the number of jobs per day
func plotMonitor(db database.BoosterwebDB) (img string, err error) {
	var byteImg bytes.Buffer
	var writerTo io.WriterTo
	var perDay map[time.Time]int
	var line *plotter.Line
	var points *plotter.Scatter

	if perDay, err = db.GetAnalysesPerDay(); err != nil {
		return
	}
	sortedDates := make([]time.Time, 0)
	for t := range perDay {
		sortedDates = append(sortedDates, t)
	}
	sort.Slice(sortedDates, func(i, j int) bool {
		return sortedDates[i].Before(sortedDates[j])
	})

	p := plot.New()
	// xticks defines how we convert and display time.Time values.
	xticks := plot.TimeTicks{Format: "2006-01-02"}

	p.Title.Text = "Number of Jobs per day"
	p.X.Label.Text = "Date"
	p.X.Tick.Marker = xticks
	p.Y.Label.Text = "Number of Jobs"
	p.Add(plotter.NewGrid())

	pts := make(plotter.XYs, len(perDay))
	for i, v := range sortedDates {
		n := perDay[v]
		pts[i].X = float64(v.Unix())
		pts[i].Y = float64(n)
		i++
	}

	if line, points, err = plotter.NewLinePoints(pts); err != nil {
		return
	}
	line.Color = color.RGBA{G: 255, A: 255}
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{R: 255, A: 255}
	p.Add(line, points)

	h := 8 * vg.Inch
	w := 8 * vg.Inch

	if writerTo, err = p.WriterTo(w, h, "svg"); err != nil {
		return
	}

	writerTo.WriteTo(&byteImg)

	img = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(byteImg.Bytes())

	return
}
