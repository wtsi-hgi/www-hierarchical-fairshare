#!/usr/bin/env perl 

# Copyright (c) 2012 Genome Research Ltd. 
# Author: Joshua C. Randall <joshua.randall@sanger.ac.uk>
#
# This file is part of www-hierarchical-fairshare.
#
# www-hierarchical-fairshare is free software: you can redistribute it
# and/or modify it under the terms of the GNU Affero General Public
# License as published by the Free Software Foundation; either version
# 3 of the License, or (at your option) any later version.  This
# program is distributed in the hope that it will be useful, but
# WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
# Affero General Public License for more details.  You should have
# received a copy of the GNU Affero General Public License along with
# this program. If not, see <http://www.gnu.org/licenses/>.

use strict;
use warnings;

use File::Spec::Functions qw(rel2abs);
use File::Basename;

print <<"EOF";
Content-Type: text/html;


<html>
  <head>
    <script type="text/javascript" src="https://www.google.com/jsapi"></script>
    <script type="text/javascript">
    google.load("visualization", "1", {packages:["treemap"]});
google.setOnLoadCallback(drawChart);
function drawChart() {
        // Create and populate the data table.
        var data = google.visualization.arrayToDataTable(
EOF

my $path = dirname(rel2abs($0));
my $data = `LSF_BINDIR="/usr/local/lsf/7.0/linux2.6-glibc2.3-x86_64/bin" LSF_ENVDIR="/etc" LSF_LIBDIR="/usr/local/lsf/7.0/linux2.6-glibc2.3-x86_64/lib" LSF_SERVERDIR="/usr/local/lsf/7.0/linux2.6-glibc2.3-x86_64/etc" $path/../lsf_tools/lsf_fairshare`;
$data =~ s/(?>\x0D\x0A?|[\x0A-\x0C\x85\x{2028}\x{2029}])//g;
print $data;


print <<"EOF";
	);

        // Create and draw the visualization.
      var tree = new google.visualization.TreeMap(document.getElementById('chart_div'));
        tree.draw(data, {
          minColor: '#fff',
          midColor: '#fdd',
          maxColor: '#f00',
          noColor: '#ddd',
    headerColor: '#aaa',
          minHighlightColor: '#0ff',
          midHighlightColor: '#0ff',
          maxHighlightColor: '#0ff',
          noHighlightColor: '#0ff',
    headerHighlightColor: '#0cc',
          headerHeight: 25,
          fontColor: 'black',
          showScale: true,
    showTooltips: true,
          maxDepth: 2,
    maxPostDepth: 2});
}
    </script>
  </head>

  <body>
    <div>
    In the following chart, priority is indicated by colour, from white (low priority) to red (high priority).
    </div>
    <div id="chart_div" style="width: 1024px; height:768px;"></div>
  </body>
</html>
EOF

