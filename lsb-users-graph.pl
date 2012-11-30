#!/usr/bin/env perl 

use strict;
use warnings;

use Data::Dumper;
use Tree::DAG_Node;

my $lsb_users_file = shift || "/usr/local/lsf/conf/lsbatch/farm2/configdir/lsb.users";

my %nodes;
my $tree_root = Tree::DAG_Node->new();
$tree_root->name("/");
$nodes{"/"} = $tree_root;

my $line = ""; 

my %usergroups; 
my %nonroot_members;    # keep track of all group members (non-root nodes)

my $lsb_users_fh;
open $lsb_users_fh, "<$lsb_users_file" or die "could not open file $lsb_users_file\n";

my $ug = 0; 
while($line = <$lsb_users_fh>) {
    if($line =~ m/^Begin\ UserGroup/) {
	$ug=1;
    } elsif($line =~ m/^End\ UserGroup/) {
	$ug=0;
    } else {
	if($ug==1) {
	    if(!( $line =~ m/(^#|^$|^GROUP_NAME)/)) {
		$line =~ m/^(.*?)[[:space:]]*\((.*?)\)[[:space:]]*\((.*)\)[[:space:]]*$/ or die "could not parse line $line\n"; 
		my $group=$1; 
		my $members=$2; 
		my $sharetuples=$3; 

		$members =~ s/^[[:space:]]+//;
		$members =~ s/[[:space:]]+$//;
		my @members = split /[[:space:]]+/,$members; 

		$sharetuples =~ s/^[[:space:]]*\[[[:space:]]*//; 
		$sharetuples =~ s/[[:space:]]*\][[:space:]]*$//; 
		my @sharetuples = split /[[:space:]]*\][[:space:]]*\[[[:space:]]*/, $sharetuples; 
		my %share; 
		foreach my $tuple (@sharetuples) {
		    my ($member, $share) = split /[[:space:]]*\,[[:space:]]*/, $tuple;
		    $share{$member} = $share;
		}
		
		foreach my $member (@members) {
		    $nonroot_members{$member} = 1;
		    if(exists($share{$member})) {
			$usergroups{$group}{$member} = $share{$member};
		    } else {
			exists($share{default}) or die "no explicit share for [$member] in group [$group] and no default set\n";
			$usergroups{$group}{$member} = $share{default};
		    }
		}
	    }
	}
    }
}
close $lsb_users_fh;


foreach my $usergroup (keys %usergroups) {
    # create tree node for usergroup
    if(!exists($nodes{$usergroup})) {
	my $node = $tree_root->new();
	$node->name($usergroup);
	$nodes{$usergroup} = $node;
    }
    
    if(!exists($nonroot_members{$usergroup})) {
	# this is a root node, connect it
	$tree_root->add_daughter($nodes{$usergroup});
	$nodes{$usergroup}->attributes->{'share'} = 1;
    }

    
    foreach my $member (keys %{$usergroups{$usergroup}}) {
	my $node_name = $member;
	if(!exists($usergroups{$member})) {
	    # this is a user, name it per group
	    $node_name = "$usergroup:$member";
	}
	
	if(!exists($nodes{$node_name})) {
	    my $node = $tree_root->new();
	    $node->name($node_name);
	    $nodes{$node_name} = $node;
	} 
	$nodes{$usergroup}->add_daughter($nodes{$node_name});
	
	$nodes{$node_name}->attributes->{'share'} = $usergroups{$usergroup}{$member};

	#print STDERR "added node $node_name to parent $usergroup with share ".$usergroups{$usergroup}{$member}."\n";
    }
}

#print Dumper($tree_root)."\n";
#print map "$_\n", @{$tree_root->draw_ascii_tree};

my @stack;
$tree_root->walk_down({
    callback => sub {
	my ($node, $options) = @_;
	
	my $share = $node->attributes->{'share'};
	my $name = $node->name;

	my $total_share_at_depth = 0;
	foreach my $sister_node ( $node->self_and_sisters() ) {
	    if(exists($sister_node->attributes->{'share'})) {
		$total_share_at_depth += $sister_node->attributes->{'share'};
	    }
	}

	my $share_ratio_at_depth = 1;
	if($total_share_at_depth > 0) {
	    $share_ratio_at_depth = $share / $total_share_at_depth;
	}

	push @{$$options{stack}}, {
	    name => $name,
	    share_ratio_at_depth => $share_ratio_at_depth,
	    total_share_at_depth => $total_share_at_depth,
	};
	return 1;
    },
    _depth => 0,
    stack => \@stack,
		      });

foreach my $share_change (@stack) {
#    print Dumper($share_change)."\n";
    my $name = $share_change->{'name'};
    if(exists($nodes{$name})) {
	$nodes{$name}->attributes->{'share_ratio_at_depth'} = $share_change->{'share_ratio_at_depth'};
	$nodes{$name}->attributes->{'total_share_at_depth'} = $share_change->{'total_share_at_depth'};
    } else {
	die "nodes{$name} does not exist";
    }
}


@stack = ();
$tree_root->walk_down({
    callback => sub {
	my ($node, $options) = @_;
	
	my $overall_share = $node->attributes->{'share_ratio_at_depth'};
	foreach my $desc_node ($node->ancestors()) {
	    my $share_ratio_at_depth = $desc_node->attributes->{'share_ratio_at_depth'};
	    $overall_share *= $share_ratio_at_depth;
	}

	my $name = $node->name;

	push @{$$options{stack}}, {
	    name => $name,
	    overall_share => $overall_share,
	};
	return 1;
    },
    _depth => 0,
    stack => \@stack,
		      });
foreach my $overall_share_change (@stack) {
#    print Dumper($share)."\n";
    my $name = $overall_share_change->{'name'};
    if(exists($nodes{$name})) {
	$nodes{$name}->attributes->{'overall_share'} = $overall_share_change->{'overall_share'};
    } else {
	die "nodes{$name} does not exist";
    }
}



print <<"EOF";
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

@stack = ();
my @rows;
push @rows, "['Group','Parent','Percentage']";
$tree_root->walk_down({
    callback => sub {
	my ($node, $options) = @_;
	
	my $name = $node->name;
	my $overall_share = $node->attributes->{'overall_share'};
	
	if($options->{_depth} == 0) {
	    # root node
	    push @rows, "['".$name."', null, $overall_share]";
	} else {
	    my $parent = $node->mother->name;
	    if(! ($name =~ m/\:/)) {
		push @rows, "['".$name."', '".$parent."', ".$overall_share."]";
	    }
	}
	
	return 1;
    },
    _depth => 0,
    stack => \@stack,
		      });
print "[".join(",\n", @rows)."]\n";


print <<"EOF";
	);

        // Create and draw the visualization.
	    var tree = new google.visualization.TreeMap(document.getElementById('chart_div'));
        tree.draw(data, {
          minColor: '#ddd',
          midColor: '#ddd',
          maxColor: '#ddd',
          noColor: '#ddd',
	  headerColor: '#aaa',
          minHighlightColor: '#0ff',
          midHighlightColor: '#0ff',
          maxHighlightColor: '#0ff',
          noHighlightColor: '#0ff',
	  headerHighlightColor: '#0cc',
          headerHeight: 25,
          fontColor: 'black',
          showScale: false,
	  showTooltips: true,
          maxDepth: 2,
	  maxPostDepth: 2});
}
    </script>
  </head>

  <body>
    <div id="chart_div" style="width: 1024px; height:768px;"></div>
  </body>
</html>
EOF
