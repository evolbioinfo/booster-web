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

package database

import (
	"github.com/fredericlemoine/booster-web/model"
)

type BoosterwebDB interface {
	GetAnalysis(id string) (*model.Analysis, error)
	UpdateAnalysis(*model.Analysis) error
	Connect() error
	Disconnect() error
	InitDatabase() error
}
