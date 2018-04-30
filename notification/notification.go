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

// This package encapsulates methods to
// notify users using smtp when the analysis finished
package notification

import (
	"fmt"
	"net/smtp"
	"regexp"
)

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Notifier interface {
	Notify(status string, analysisId string, runName string, workflow string, email string) error
}

type EmailNotifier struct {
	server    string // smtp server
	port      int    // smtp port
	user      string // smtp user
	pass      string // smtp password
	sender    string // Sender Email
	resulturl string // url to the result page
}
type NullNotifier struct {
}

func NewEmailNotifier(smtp string, port int, user, pass, sender, resultpage string) (notifier *EmailNotifier) {
	return &EmailNotifier{
		server:    smtp,
		port:      port,
		user:      user,
		pass:      pass,
		sender:    sender,
		resulturl: resultpage,
	}
}

func NewNullNotifier() (notifier *NullNotifier) {
	return &NullNotifier{}
}

func (n *NullNotifier) Notify(status string, analysisId string, runName string, workflow string, email string) (err error) {
	return
}

// Workflow may be FastTree or PhyML-SMS
func (n *EmailNotifier) Notify(status string, analysisId string, runName string, workflow string, email string) (err error) {
	// Connect to the remote SMTP server.
	if email != "" && n.server != "" && n.user != "" && n.pass != "" && n.sender != "" && validateEmail(email) {
		ref := "Lemoine, F., Domelevo-Entfellner, J.-B., Wilkinson, E., Correia, D., Davila Felipe, M., De Oliveira, T., Gascuel, O. (2018). Renewing Felsenstein's Phylogenetic Bootstrap in the Era of Big Data, Nature 556, 452-45."
		jobstr := "Booster[1]"
		if workflow == "PhyML-SMS" {
			ref = "[1] Lefort, V., Longueville, J. E., & Gascuel, O. (2017). SMS: Smart Model Selection in PhyML. Molecular Biology and Evolution.\n[2] " + ref
			jobstr = "PhyML-SMS[1]+Booster[2]"
		} else if workflow == "FastTree" {
			ref = "[1] Price, M. N., Dehal, P. S., & Arkin, A. P. (2009). FastTree: computing large minimum evolution trees with profiles instead of a distance matrix. Molecular biology and evolution, 26(7), 1641-1650.\n[2] " + ref
			jobstr = "FastTree[1]+Booster[2]"
		} else {
			ref = "[1] " + ref
		}

		runnamestr := " "
		if runName != "" {
			runnamestr = fmt.Sprintf(" (run name: %s) ", runName)
		}

		auth := smtp.PlainAuth("", n.user, n.pass, n.server)
		body := fmt.Sprintf("Dear booster-web user, \n\nYour job %s%sis done (status : '%s').\nResults are available at the following page:\n%s/%s\n\nBest regards,\n\nThe BOOSTER-WEB team\nEvolutionary Biology Unit - USR 3756 Institut Pasteur - CNRS\nhttps://research.pasteur.fr/en/team/evolutionary-bioinformatics\n\n%s", jobstr, runnamestr, status, n.resulturl, analysisId, ref)
		msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: booster-web results\n\n%s", n.sender, email, body)

		err = smtp.SendMail(fmt.Sprintf("%s:%d", n.server, n.port), auth, n.sender, []string{email}, []byte(msg))
	}
	return
}

func validateEmail(email string) bool {
	return emailRegexp.MatchString(email)
}
