package api

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDecompress(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		output      string
		shouldError bool
	}{
		{
			desc:        "Empty",
			input:       "",
			output:      "",
			shouldError: true,
		},
		{
			desc:        "Single |",
			input:       "|",
			output:      "",
			shouldError: true,
		},
		{
			desc:        "Minimal Valid",
			input:       "||",
			output:      "",
			shouldError: false,
		},
		{
			desc:        "No compression",
			input:       "||I love love slay the relics and slay the spire",
			output:      "I love love slay the relics and slay the spire",
			shouldError: false,
		},
		{
			desc:        "Unused compression",
			input:       "Foo|Bar||I love love slay the relics and slay the spire",
			output:      "I love love slay the relics and slay the spire",
			shouldError: false,
		},
		{
			desc:        "Basic compression",
			input:       "love|slay the||I &0 &0 &1 relics and &1 spire",
			output:      "I love love slay the relics and slay the spire",
			shouldError: false,
		},
		{
			desc:        "Nested compression succeeds",
			input:       "love|the|slay &1||I &0 &0 &2 relics and &2 spire",
			output:      "I love love slay the relics and slay the spire",
			shouldError: false,
		},
		{
			desc:        "Interleaved compression succeeds",
			input:       "AAA|BBB&|0CCC||&1&2",
			output:      "BBBAAACCC",
			shouldError: false,
		},
		{
			desc:        "Interleaved repeated compression fails",
			input:       "AAA|BBB&|1CCC||&1&2",
			output:      "BBB&1CCC",
			shouldError: false,
		},
		{
			desc:        "Small Deck",
			input:       "card|junk||0,1,1,0,2,0;;;&01;&1;x;;&02;&1;y;;&03;&1;z",
			output:      "0,1,1,0,2,0;;;card1;junk;x;;card2;junk;y;;card3;junk;z",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualOutput, err := decompress(tc.input)
			if tc.shouldError {
				assert.Equal(t, true, err != nil)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, actualOutput, tc.output)
		})
	}
}

func TestParseCommaDelimitedIntegerArray(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		output      []int
		shouldError bool
	}{
		{
			desc:        "Empty",
			input:       "",
			output:      []int{},
			shouldError: false,
		},
		{
			desc:        "Dash",
			input:       "-",
			output:      []int{},
			shouldError: false,
		},
		{
			desc:        "Invalid",
			input:       "{",
			output:      []int{},
			shouldError: true,
		},
		{
			desc:        "Single",
			input:       "19",
			output:      []int{19},
			shouldError: false,
		},
		{
			desc:        "Many",
			input:       "1,2,3,4,5,6,7,8,9,1,2,3,4,5,6,7,8,9",
			output:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualOutput, err := parseCommaDelimitedIntegerArray(tc.input)
			if tc.shouldError {
				assert.Equal(t, true, err != nil)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, len(actualOutput), len(tc.output))
			for i := range actualOutput {
				assert.Equal(t, actualOutput[i], tc.output[i])
			}
		})
	}
}

func TestDecompressDeck(t *testing.T) {
	testCases := []struct {
		desc        string
		input       string
		output      map[string]int
		shouldError bool
	}{
		{
			desc:        "Empty",
			input:       "",
			output:      map[string]int{},
			shouldError: true,
		},
		{
			desc:        "Invalid card index",
			input:       "card|junk||};;;&01;&1;x;;&02;&1;y;;&03;&1;z",
			output:      map[string]int{},
			shouldError: true,
		},
		{
			desc:        "Negative card index",
			input:       "card|junk||-1;;;&01;&1;x;;&02;&1;y;;&03;&1;z",
			output:      map[string]int{},
			shouldError: true,
		},
		{
			desc:        "Out of bounds card index",
			input:       "card|junk||3;;;&01;&1;x;;&02;&1;y;;&03;&1;z",
			output:      map[string]int{},
			shouldError: true,
		},
		{
			desc:  "Simple deck",
			input: "card|junk||0,1,1,0,2,0;;;&01;&1;x;;&02;&1;y;;&03;&1;z",
			output: map[string]int{
				"card1": 3,
				"card2": 2,
				"card3": 1,
			},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualOutput, err := decompressDeck(tc.input)
			if tc.shouldError {
				assert.Equal(t, true, err != nil)
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, len(actualOutput), len(tc.output))
			for i := range actualOutput {
				assert.Equal(t, actualOutput[i], tc.output[i])
			}
		})
	}
}

// getBigDeckString makes us a 52 card compressed deck string for testing with.
func getBigDeckString() string {
	compressionDict := make([]string, 0, 128)

	// Generate cards.
	compressedCards := make([]string, 0, 52)
	for i := 0; i < 52; i++ {
		compressionDict = append(compressionDict, fmt.Sprintf("Card Name %d;other details;junk", i))
		compressedCards = append(compressionDict, fmt.Sprintf("%c", WILDCARDS[i]))
	}
	cardsPart := strings.Join(compressedCards, ";;")

	// Generate deck.
	deckIndices := make([]string, 0, 100)
	for i := 0; i < 52; i++ {
		deckIndices = append(deckIndices, fmt.Sprintf("%d", i))
	}
	for i := 0; i < 48; i++ {
		deckIndices = append(deckIndices, "17")
	}
	deckPart := strings.Join(deckIndices, ",")

	// Put it all together.
	compressionPart := strings.Join(compressionDict, "|")
	compressedPart := fmt.Sprintf("%s;;;%s", deckPart, cardsPart)
	return fmt.Sprintf("%s||%s", compressionPart, compressedPart)
}

func TestDecompressBigDeck(t *testing.T) {
	output, err := decompressDeck(getBigDeckString())
	assert.NilError(t, err)

	assert.Equal(t, len(output), 52)
	for card, count := range output {
		switch card {
		case "Card Name 17":
			assert.Equal(t, count, 49, card)
		default:
			assert.Equal(t, count, 1, card)
		}
	}
}

func BenchmarkDecompressDeck(b *testing.B) {
	testCases := []struct {
		desc  string
		input string
	}{
		{
			desc:  "Simple deck",
			input: "card|junk||0,1,1,0,2,0;;;&01;&1;x;;&02;&1;y;;&03;&1;z",
		},
		{
			desc:  "Big deck",
			input: getBigDeckString(),
		},
	}

	b.ResetTimer()
	for _, tc := range testCases {
		b.Run(tc.desc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				decompressDeck(tc.input)
			}
		})
	}
}

var searingBlows = "damage|#yUpgraded|cards|Deal|Searing|your|.;;|number|Discard|#yBlock|times|#yExhaust|from|discard|hand|card|Unplayable|Gain|additional|deal|Whenever|#yWrath|removed|Can|#yShivs|any|13,13,13,13,13,13,13,13,13,13|&0. NL &n be &1 &p NL &7 of|NL &c &5 &d pile NL into &5|;0;-;-1;-1;+;0;|&q,&q,&q,&q,&q,&q,&q|&0. NL Put all cost 0|;0;-;-1;-1;|,11,11,11,11,|,12,12,12,|. NL &8 1 |;2;2;0;|;1;1;|&r &a.;7&B3| &0. NL |;&3 ||0,1,1,1,1,1,2,2,2,2,2,3,4,5,5,6,6,7,8,9,10,10,10,10,10,10&y11,11&z12,12,12,&v,&q,&q,&q;;;Ascender's Bane&x-;0;-;-;0,1;-2;-2;4;1;5;#y&g. NL #yEthereal. NL &nnot be &m NL &c &5 deck&6Strike&t-&F#g9 &0.;-&C0;0;1&F6 &0&6Defend&t2;&h #g8 &9.;2&C1;0;1;&h 5 &9&6Survivor&t2;&h #g11 &9&A&f.;2&C1;0;1;&h 8 &9&A&f&6Neutralize&t3&F#g4&EApply #g2 #yWeak.;3;0;0;0;0;1&F3&EApply 1 #yWeak&6All for One&t-&F#g14 &w &2 &s &e.;-&B4;2&F10 &w &2 &s &e&6All for One+&x_;1;-;_;-&B4;2&F14 &w &2 &s &e&6Accuracy;0;-;14;14;+;0;-;&o &j #g6 NL &i &0.;-&C2;3;1;&o &j 4 NL &i &0&6Acrobatics&t-;Draw #g4 &2&A&f.;-&C1;2;1;Draw 3 &2&A&f&6Rushdown&t4;&k you enter NL &l, draw 2 &2.;4;1;0;2;3;3;&k you enter NL &l, draw 2 &2&6Adrenaline+&x_;1;5,6;_;5,6;0;0;1;4;1;&h [E] [E]. NL Draw 2 &2. NL &b&6&4 Blow+3&x&4 Blow+4;3;_&F#g34 &D;0&F27 &r &a&6&4 Blow+99&x&4 Blow+100;99;_&F#g5362 &D;0&F5259 &r &a&6&4 Blow+2000&x&4 Blow+2001;2000;_&F#g2009016 &D;0&F2007012 &r &a&6Shiv&t6&F#g6&E&b.;6;0;0;0;1;4&F4&E&b&6;&g;&g &2 cannot be played &c &5 &e&6Ethereal;If this &f is in &5 &e at the end of turn, it is exhausted. Exhausted &2 are &m &c &5 deck until the end of combat&6Block;Until next turn, prevents &0&6Weak;Weakened creatures &j #b25% less &0 with Attacks&6Wrath;In this #yStance, you &j and receive double attack &0&6[E] Energy;Energy is used to play &2 &c &5 &e&6Exhaust;Removed until end of combat&6Upgrade;Upgrading &2 makes them more powerful. Cards can only be upgraded once."
var realDeck = "damage|card|your|Deal|enemies|.;;|cards|Whenever|discard|pile|draw|hand|Attack|into|#yVulnerable|from|#yExhausted|Unplayable|Vulnerable|combat|played|#yExhaust|creatures|gain|#yStatus|#yDazed|Shuffle|removed|ALL|#yCurse|#yBlock|Attacks|the enemy has NL &e, NL &n [E] and NL|;0;-;-1;-1;+;0;-;&3 |;Choose an &c or NL Power &1. Add | copy of this &1 NL &d &2 dis&1 |Can only be &k if NL every &1 in|.;-;1;1;0;2;0;&3 |NL dis&1 &9 on top of NL &2|;0;-;-1;-1;+;0;| #yWeak to &s NL &4. NL |NL &o or &t &1, NL deal|&1 is NL &g, NL &n|NL &b is an &c. NL|&0. NL Put a &1 &f| &0 to &s NL |.;-;0;0;0;|&0. NL If &x &a| at the end of | of that &1 &d |&7 you &a a &G|&0. NL &q a &p| &0. NL Draw | &0. NL Add |&B &2 &I &3|&J &2 &D &a|&d NL &2 &a| &1s are |;1;1;| &f &2 |;0;-;|;3;0;| &0 | until ||0,1,2,2,3,4,5,6,7,8,9,10,0,11,9;;;Anger&y#g8&Sa&A&9&L2;0;&3 6&Sa&A&9&5Pommel Strike&y#g10&R#g2 &1s&C9&R1 &1&5Twin Strike&y#g7&[twice&C5&[twice&5Intimidate&E0,1;Apply #g2&F&l.;0,1;0;0;1&`Apply 1&F&l&5Fire Breathing&E2,3;&P #g10&K&4.;2,3&Y2&`&P 6&K&4&5Dual Wield&E-&z2 NL copies&ONL &2 &b.;-&Y1;3;0&za copy NL&O&2 NL &b&5Headbutt&y#g12 &V &9&C9 &V &9&5Dropkick&E4,5;&3 #g8 &M 1 &1.;4,5&Y0&`&3 5 &M 1 &1&5Feel No Pain&E1,6;&7 a &H #g4 &v.;1,6&Y2&`&7 a &H 3 &v&5Cleave&y#g11&K&4&C8&K&4&5Clash&E-;&T #g18 &0&L2;0;&T 14 &0&5Reckless Charge&_12;12;+&_&3 #g10 &Q &W &9&L3;0;&3 7 &Q &W &9&5Dazed&_-1;-1;-&_-;7,8;-2;-2;3;2;4;#y&h. NL #yEthereal&5;Weak;Weakened &m deal #b25% less&[with &cs&5Exhaust;Removed&]end of &j&5Status;Status&X&r&N&j&5Curse;Curse&Xnegative &1s that stay in &2 deck&5&i;&i &m take #b50% more&[&f &cs&5[E] Energy;Energy is used to play &1s&Z&b&5Block;Until next turn, prevents &0&5&h;&h &1s cannot be &k&Z&b&5Ethereal;If this &1 is in &2 &b&Nturn, it is exhausted. Exhausted&X&r&Zdeck&]the end of &j."
var megaDeckOld = "damage|#yBlock|#yExhaust|Deal|your|.;;|card|Gain|cards|#yVulnerable|#yPoison|Whenever|discard|Apply|turn|this|additional|enemies|hand|pile|combat|#yLightning|Draw|gain|random|into|played|the|from|#yChannel|times|draw|#yWeak|Strike|Attack|you|enemy|start|#yRetain|increase|Energy|Double|#yEthereal|attack|Attacks|Discard|twice|each|discarded|creatures|#yScry|number|play|#yStance|#yIntangible|have|#yShivs|[E]|receive|#yDexterity|Permanently|Costs|apply|equal|first|for|ALL|Unplayable|Poison|#yStrength|next|Defend|lose|#yUpgraded|Vulnerable|#g4|#yUpgrade|unblocked|Dexterity|Put|#g2|#g8|Form|#g10|and|Strength|copy|only||0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,33,34,35,36,33,37,18,38,39,40,41,42,37,39,43,44,45,46,47,48,49,50,51,25,29,52,53,54,49,55,56,57,57,57,57,57,57,57,57,57,57,57,57,57,57,57,58,58,58,59,59,59,60,60,60,61,61,61,62,62,62,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,36,36,36;;;Demon &$;0;-;-1;-1;+;0;0;At &r &C of &4 NL &e, &n #g3 &>.;0;3;3;2;4;0;At &r &C of &4 NL &e, &n 2 &>&5Wraith &$;0;-;-1;-1;+;0;1,2;&7 #g3 &T. NL At &r end of &4 NL &e, &- 1 &Z.;1,2;3;3;2;4;1;&7 2 &T. NL At &r end of &4 NL &e, &- 1 &Z&5Echo &$;0;-;-1;-1;+;0;-;The &/ &6 &A &R NL &M &e is &q NL &L.;3;3;3;2;4;2;#yE&rreal. NL The &/ &6 &A &R NL &M &e is &q NL &L&5Deva &$;0;-;-1;-1;+;0;4;At &r &C of &4 NL &e, &n &X NL &' &E &f &n NL by !M!.;3,4;3;3;2;4;3;#yE&rreal. NL At &r &C of &4 NL &e, &n &X NL &' &E &f &n NL by !M!&5Ascender's Bane;0;-;-1;-1;-;0;-;-;5,3;-2;-2;4;1;5;#yUn&Rable. NL #yE&rreal. NL Cannot be removed NL &s &4 deck&5Bash;0;-;-1;-1;+;0;6;&3 &! &0. NL &d #g3 &9.;6;2;2;0;0;0;&3 8 &0. NL &d 2 &9&5Survivor;0;-;-1;-1;+;0;7;&7 #g11 &1. NL Dis&6 1 &6.;7;1;1;1;0;1;&7 8 &1. NL Dis&6 1 &6&5Zap;0;-;-1;-1;+;0;8,9;&t 1 &l.;8,9;1;0;1;0;2;&t 1 &l&5Eruption;0;-;-1;-1;+;0;10;&3 9 &0. NL Enter #yWrath.;10;2;1;0;0;3;&3 9 &0. NL Enter #yWrath&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;0;&3 6 &0&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;1;&3 6 &0&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;2;&3 6 &0&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;0;&7 5 &1&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;1;&7 5 &1&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;3;&7 5 &1&5Wreath of Flame;0;-;-1;-1;+;0;-;Your &< &z deals NL &# &g &0.;-;1;1;1;3;3;Your &< &z deals NL 5 &g &0&5Wheel Kick;0;-;-1;-1;+;0;-;&3 &)0 &0. NL &m 2 &6s.;-;2;2;0;3;3;&3 15 &0. NL &m 2 &6s&5Com&j Driver;0;-;-1;-1;+;0;-;&3 &! &0. NL &m 1 &6 &^ &M NL unique Orb &A &V.;-;1;1;0;2;2;&3 7 &0. NL &m 1 &6 &^ &M NL unique Orb &A &V&5Consecrate;0;-;-1;-1;+;0;-;&3 &# &0 to &% NL &h.;-;0;0;0;2;3;&3 5 &0 to &% NL &h&5Auto-Shields;0;-;-1;-1;+;0;7;If &A &V no &1, NL &n #g15 &1.;7;1;1;1;3;2;If &A &V no &1, NL &n 11 &1&5Genetic Algorithm;0;-;-1;-1;+;0;7,11;&7 1 &1. NL &_ &E NL &f &6's &1 by !M!. NL &2.;7,11;1;1;1;3;2;&7 1 &1. NL &_ &E NL &f &6's &1 by !M!. NL &2&5Eviscerate;0;-;-1;-1;+;0;4;&` 1 less &X NL &^ &M &6 dis&6ed NL &f &e. NL &3 #g9 &0 3 &v.;4;3;3;0;3;1;&` 1 less &X NL &^ &M &6 dis&6ed NL &f &e. NL &3 7 &0 3 &v&5Bouncing Flask;0;-;-1;-1;+;0;12;&d 3 &a to a NL &o &B &: NL &v.;12;2;2;1;3;1;&d 3 &a to a NL &o &B 3 NL &v&5Twin &y;0;-;-1;-1;+;0;-;&3 #g7 &0 &L.;-;1;1;0;2;0;&3 5 &0 &L&5Static Discharge;0;-;-1;-1;+;0;8,9;&b &A &Y NL &, &I NL &0, &t &) NL &l.;8,9;1;1;2;3;2;&b &A &Y NL &, &I NL &0, &t 1 NL &l&5Noxious Fumes;0;-;-1;-1;+;0;12;At &r &C of &4 NL &e, &[ #g3 &a to NL &% &h.;12;1;1;2;3;1;At &r &C of &4 NL &e, &[ 2 &a to NL &% &h&5Terror;0;-;-1;-1;+;0;6,11;&d 99 &9. NL &2.;6,11;1;0;1;3;1;&d 99 &9. NL &2&5Nirvana;0;-;-1;-1;+;0;13,7;&b &A &P, NL &n &: &1.;13,7;1;1;2;3;3;&b &A &P, NL &n 3 &1&5Pommel &y;0;-;-1;-1;+;0;-;&3 &! &0. NL &m &) &6s.;-;1;1;0;2;0;&3 9 &0. NL &m 1 &6&5Blade Dance;0;-;64;64;+;0;-;Add &: &W &p &4 NL &i.;-;1;1;1;2;1;Add 3 &W &p &4 NL &i&5Entrench;0;-;-1;-1;+;0;7;&G &4 &1.;7;2;1;1;3;0;&G &4 &1&5Rebound;0;-;-1;-1;+;0;-;&3 #g12 &0. NL &( &r &< &6 &A NL &R &f &e on top of NL &4 &w &j.;-;1;1;0;2;2;&3 9 &0. NL &( &r &< &6 &A NL &R &f &e on top of NL &4 &w &j&5Heatsinks;0;-;-1;-1;+;0;-;&b &A &R a NL Power &6, &w &) NL &6s.;-;1;1;2;3;2;&b &A &R a NL Power &6, &w 1 NL &6&5Endless Agony;0;-;-1;-1;+;0;11;&3 #g6 &0. NL &b &A &w NL &f &6, add a &} of NL it &p &4 &i. NL &2.;11;0;0;0;3;1;&3 4 &0. NL &b &A &w NL &f &6, add a &} of NL it &p &4 &i. NL &2&5Cloak &' Dagger;0;-;64;64;+;0;7;&7 6 &1. NL Add &) &W &p &4 NL &i.;7;1;1;1;2;1;&7 6 &1. NL Add 1 #yShiv &p &4 NL &i&5Protect;0;-;-1;-1;+;0;14,7;&D. NL &7 #g16 &1.;14,7;2;2;1;2;3;&D. NL &7 12 &1&5Go &^ &r Eyes;0;-;-1;-1;+;0;15;&3 &: &0. NL If &r &B intends to NL &I, &[ &) &x.;15;0;0;0;2;2;&3 3 &0. NL If &r &B intends to NL &I, &[ 1 &x&5Masterful Stab;0;-;-1;-1;+;0;4;&` 1 &g &X NL &^ &M time &A &- NL HP &f &k. NL &3 #g16 &0.;4;0;0;0;3;1;&` 1 &g &X NL &^ &M time &A &- NL HP &f &k. NL &3 12 &0&5Empty Fist;0;-;-1;-1;+;0;16;&3 #g14 &0. NL Exit &4 &S.;16;1;1;0;2;3;&3 9 &0. NL Exit &4 &S&5Backflip;0;-;-1;-1;+;0;7;&7 &# &1. NL &m 2 &6s.;7;1;1;1;2;1;&7 5 &1. NL &m 2 &6s&5Third Eye;0;-;-1;-1;+;0;7,13;&7 #g9 &1. NL &P !M!.;7,13;1;1;1;2;3;&7 7 &1. NL &P !M!&5Recycle;0;-;-1;-1;+;0;11,4;&2 a &6. NL &7 &X &] to its NL cost.;11,4;1;0;1;3;2;&2 a &6. NL &7 &X &] to its NL cost&5&G &F;0;-;-1;-1;+;0;11;&G &4 &F. NL &2.;11;1;0;1;3;2;&G &4 &F. NL &2&5Headbutt;0;-;-1;-1;+;0;-;&3 #g12 &0. NL &( a &6 &s &4 NL dis&6 &j on top of NL &4 &w &j.;-;1;1;0;2;0;&3 9 &0. NL &( a &6 &s &4 NL dis&6 &j on top of NL &4 &w &j&5Uppercut;0;-;-1;-1;+;0;15,6;&3 13 &0. NL &d &) &x. NL &d &) &9.;15,6;2;2;0;3;0;&3 13 &0. NL &d 1 &x. NL &d 1 &9&5Stack;0;-;-1;-1;+;0;7;&7 &1 &] to &r NL &Q of &6s in NL &4 dis&6 &j +3.;7;1;1;1;2;2;&7 &1 &] to &r NL &Q of &6s in NL &4 dis&6 &j&5Deadly &@;0;-;-1;-1;+;0;12;&d #g7 &a.;12;1;1;1;2;1;&d 5 &a&5Shrug It Off;0;-;-1;-1;+;0;7;&7 #g11 &1. NL &m 1 &6.;7;1;1;1;2;0;&7 8 &1. NL &m 1 &6&5Clash;0;-;-1;-1;+;0;-;Can &~ be &q if NL every &6 in &4 NL &i is an &z. NL &3 #g18 &0.;-;0;0;0;2;0;Can &~ be &q if NL every &6 in &4 NL &i is an &z. NL &3 14 &0&5Flying Sleeves;0;-;-1;-1;+;0;14;&D. NL &3 #g6 &0 &L.;14;1;1;0;2;3;&D. NL &3 4 &0 &L&5Follow-Up;0;-;-1;-1;+;0;4;&3 #g11 &0. NL If &r last &6 &q NL &f &k was an NL &z, &n &X.;4;1;1;0;2;3;&3 7 &0. NL If &r last &6 &q NL &f &k was an NL &z, &n &X&5Acrobatics;0;-;-1;-1;+;0;-;&m &: &6s. NL Dis&6 1 &6.;-;1;1;1;2;1;&m 3 &6s. NL Dis&6 1 &6&5Anger;0;-;-1;-1;+;0;-;&3 &# &0. NL Add a &} of &f &6 NL &p &4 dis&6 &j.;-;0;0;0;2;0;&3 6 &0. NL Add a &} of &f &6 NL &p &4 dis&6 &j&5Iron Wave;0;-;-1;-1;+;0;7;&7 #g7 &1. NL &3 #g7 &0.;7;1;1;0;2;0;&7 5 &1. NL &3 5 &0&5Hologram;0;-;-1;-1;+;0;7;&7 #g5 &1. NL &( a &6 &s &4 NL dis&6 &j &p &4 NL &i.;7,11;1;1;1;2;2;&7 3 &1. NL &( a &6 &s &4 NL dis&6 &j &p &4 NL &i. NL &2&5Deflect;0;-;-1;-1;+;0;7;&7 #g7 &1.;7;0;0;1;2;1;&7 4 &1&5Sword Boomerang;0;-;-1;-1;+;0;-;&3 3 &0 to a NL &o &B &: NL &v.;-;1;1;0;2;0;&3 3 &0 to a NL &o &B 3 NL &v&5Crippling Cloud;0;-;-1;-1;+;0;12,15,11;&d #g7 &a &' 2 NL &x to &% &h. NL &2.;12,15,11;2;2;1;3;1;&d 4 &a &' 2 NL &x to &% &h. NL &2&5Violence;0;-;-1;-1;+;0;11;&( &: &o &zs NL &s &4 &w &j NL &p &4 &i. NL &2.;11;0;0;1;4;4;&( 3 &o &zs NL &s &4 &w &j NL &p &4 &i. NL &2&5H&' of Greed;0;-;-1;-1;+;0;17;&3 &)5 &0. NL If #yFatal, &n &)5 Gold.;17;2;2;0;4;4;&3 20 &0. NL If #yFatal, &n 20 Gold&5Swift &y;0;-;-1;-1;+;0;-;&3 &! &0.;-;0;0;0;3;4;&3 7 &0&5Apo&rosis;0;-;-1;-1;+;0;18,11;&; &% &4 NL &6s &^ &r rest of NL &k. NL &2.;18,11;2;1;1;4;4;&; &% &4 NL &6s &^ &r rest of NL &k. NL &2&5Master of Strategy;0;-;-1;-1;+;0;11;&m &: &6s. NL &2.;11;0;0;1;4;4;&m 3 &6s. NL &2&5Searing Blow+2000;0;-;-1;-1;Searing Blow+2001;2000;_;&3 &)009016 &0. NL Can be &+ any NL &Q of &v.;18;2;2;0;3;0;&3 2007012 &0. NL Can be &+ any NL &Q of &v&5Shiv;0;-;-1;-1;+;0;11;&3 #g6 &0. NL &2.;11;0;0;0;1;4;&3 4 &0. NL &2&5;&{;&{ adds &g &0 to &Is&5Intangible;Reduce &% &0 taken &' HP loss to #b1&5&.;&. improves Block &ned &s &6s&5E&rreal;If &f &6 is in &4 &i at &r end of &e, it is exhausted. Exhausted &6s are removed &s &4 deck until &r end of &k&5&X &F;&F is used to &R &6s &s &4 &i&5Un&Rable;Un&Rable &6s cannot be &q &s &4 &i&5&*;&* &O take #b50% more &0 &s &zs&5Block;Until &< &e, prevents &0&5Channel;Channeling an Orb puts it &p &4 &/ empty slot. If &A &V no empty slots, &4 &/ Orb is automatically #yEvoked to make room&5Lightning;Orb: &3s &0 to &o &h&5Wrath;In &f &S, &A deal &' &Y double &I &0&5Exhaust;Removed until end of &k&5&@;&@ed &O &- HP at &r &C of &rir &e. Each &e, &@ is reduced by #b1&5Scry;Look at &r top X &6s of &4 &w &j. You may dis&6 any of &rm&5Retain;Not dis&6ed at &r end of &4 &e&5Weak;Weakened &O deal #b25% less &0 with &zs&5Stance;You can &~ &V one stance at a time&5Fatal;Triggers whenever &f &6 kills a non-minion &B&5Upgrade;Upgrading &6s makes &rm more powerful. Cards can &~ be upgraded once."
var megaDeckNew = "damage|#yBlock|#yExhaust|Deal|your|.;;|card|Gain|cards|#yVulnerable|#yPoison|Whenever|discard|Apply|turn|this|additional|enemies|hand|pile|combat|#yLightning|Draw|gain|random|into|played|the|from|#yChannel|times|draw|#yWeak|Strike|Attack|you|enemy|start|#yRetain|increase|Energy|Double|#yEthereal|attack|Attacks|Discard|twice|each|discarded|creatures|#yScry|number|play|#yStance|#yIntangible|have|#yShivs|[E]|receive|#yDexterity|Permanently|Costs|apply|equal|first|for|ALL|Unplayable|Poison|#yStrength|next|Defend|lose|#yUpgraded|Vulnerable|#g4|#yUpgrade|unblocked|Dexterity|Put|#g2|#g8|Form|#g10|and|Strength|copy|only||0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,33,34,35,36,33,37,18,38,39,40,41,42,37,39,43,44,45,46,47,48,49,50,51,25,29,52,53,54,49,55,56,57,57,57,57,57,57,57,57,57,57,57,57,57,57,57,58,58,58,59,59,59,60,60,60,61,61,61,62,62,62,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63,63;;;Demon &$;0;-;-1;-1;+;0;0;At &r &C of &4 NL &e, &n #g3 &>.;0;3;3;2;4;0;At &r &C of &4 NL &e, &n 2 &>&5Wraith &$;0;-;-1;-1;+;0;1,2;&7 #g3 &T. NL At &r end of &4 NL &e, &- 1 &Z.;1,2;3;3;2;4;1;&7 2 &T. NL At &r end of &4 NL &e, &- 1 &Z&5Echo &$;0;-;-1;-1;+;0;-;The &/ &6 &A &R NL &M &e is &q NL &L.;3;3;3;2;4;2;#yE&rreal. NL The &/ &6 &A &R NL &M &e is &q NL &L&5Deva &$;0;-;-1;-1;+;0;4;At &r &C of &4 NL &e, &n &X NL &' &E &f &n NL by !M!.;3,4;3;3;2;4;3;#yE&rreal. NL At &r &C of &4 NL &e, &n &X NL &' &E &f &n NL by !M!&5Ascender's Bane;0;-;-1;-1;-;0;-;-;5,3;-2;-2;4;1;5;#yUn&Rable. NL #yE&rreal. NL Cannot be removed NL &s &4 deck&5Bash;0;-;-1;-1;+;0;6;&3 &! &0. NL &d #g3 &9.;6;2;2;0;0;0;&3 8 &0. NL &d 2 &9&5Survivor;0;-;-1;-1;+;0;7;&7 #g11 &1. NL Dis&6 1 &6.;7;1;1;1;0;1;&7 8 &1. NL Dis&6 1 &6&5Zap;0;-;-1;-1;+;0;8,9;&t 1 &l.;8,9;1;0;1;0;2;&t 1 &l&5Eruption;0;-;-1;-1;+;0;10;&3 9 &0. NL Enter #yWrath.;10;2;1;0;0;3;&3 9 &0. NL Enter #yWrath&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;0;&3 6 &0&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;1;&3 6 &0&5&y;0;-;-1;-1;+;0;-;&3 #g9 &0.;-;1;1;0;0;2;&3 6 &0&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;0;&7 5 &1&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;1;&7 5 &1&5&=;0;-;-1;-1;+;0;7;&7 &# &1.;7;1;1;1;0;3;&7 5 &1&5Wreath of Flame;0;-;-1;-1;+;0;-;Your &< &z deals NL &# &g &0.;-;1;1;1;3;3;Your &< &z deals NL 5 &g &0&5Wheel Kick;0;-;-1;-1;+;0;-;&3 &)0 &0. NL &m 2 &6s.;-;2;2;0;3;3;&3 15 &0. NL &m 2 &6s&5Com&j Driver;0;-;-1;-1;+;0;-;&3 &! &0. NL &m 1 &6 &^ &M NL unique Orb &A &V.;-;1;1;0;2;2;&3 7 &0. NL &m 1 &6 &^ &M NL unique Orb &A &V&5Consecrate;0;-;-1;-1;+;0;-;&3 &# &0 to &% NL &h.;-;0;0;0;2;3;&3 5 &0 to &% NL &h&5Auto-Shields;0;-;-1;-1;+;0;7;If &A &V no &1, NL &n #g15 &1.;7;1;1;1;3;2;If &A &V no &1, NL &n 11 &1&5Genetic Algorithm;0;-;-1;-1;+;0;7,11;&7 1 &1. NL &_ &E NL &f &6's &1 by !M!. NL &2.;7,11;1;1;1;3;2;&7 1 &1. NL &_ &E NL &f &6's &1 by !M!. NL &2&5Eviscerate;0;-;-1;-1;+;0;4;&` 1 less &X NL &^ &M &6 dis&6ed NL &f &e. NL &3 #g9 &0 3 &v.;4;3;3;0;3;1;&` 1 less &X NL &^ &M &6 dis&6ed NL &f &e. NL &3 7 &0 3 &v&5Bouncing Flask;0;-;-1;-1;+;0;12;&d 3 &a to a NL &o &B &: NL &v.;12;2;2;1;3;1;&d 3 &a to a NL &o &B 3 NL &v&5Twin &y;0;-;-1;-1;+;0;-;&3 #g7 &0 &L.;-;1;1;0;2;0;&3 5 &0 &L&5Static Discharge;0;-;-1;-1;+;0;8,9;&b &A &Y NL &, &I NL &0, &t &) NL &l.;8,9;1;1;2;3;2;&b &A &Y NL &, &I NL &0, &t 1 NL &l&5Noxious Fumes;0;-;-1;-1;+;0;12;At &r &C of &4 NL &e, &[ #g3 &a to NL &% &h.;12;1;1;2;3;1;At &r &C of &4 NL &e, &[ 2 &a to NL &% &h&5Terror;0;-;-1;-1;+;0;6,11;&d 99 &9. NL &2.;6,11;1;0;1;3;1;&d 99 &9. NL &2&5Nirvana;0;-;-1;-1;+;0;13,7;&b &A &P, NL &n &: &1.;13,7;1;1;2;3;3;&b &A &P, NL &n 3 &1&5Pommel &y;0;-;-1;-1;+;0;-;&3 &! &0. NL &m &) &6s.;-;1;1;0;2;0;&3 9 &0. NL &m 1 &6&5Blade Dance;0;-;64;64;+;0;-;Add &: &W &p &4 NL &i.;-;1;1;1;2;1;Add 3 &W &p &4 NL &i&5Entrench;0;-;-1;-1;+;0;7;&G &4 &1.;7;2;1;1;3;0;&G &4 &1&5Rebound;0;-;-1;-1;+;0;-;&3 #g12 &0. NL &( &r &< &6 &A NL &R &f &e on top of NL &4 &w &j.;-;1;1;0;2;2;&3 9 &0. NL &( &r &< &6 &A NL &R &f &e on top of NL &4 &w &j&5Heatsinks;0;-;-1;-1;+;0;-;&b &A &R a NL Power &6, &w &) NL &6s.;-;1;1;2;3;2;&b &A &R a NL Power &6, &w 1 NL &6&5Endless Agony;0;-;-1;-1;+;0;11;&3 #g6 &0. NL &b &A &w NL &f &6, add a &} of NL it &p &4 &i. NL &2.;11;0;0;0;3;1;&3 4 &0. NL &b &A &w NL &f &6, add a &} of NL it &p &4 &i. NL &2&5Cloak &' Dagger;0;-;64;64;+;0;7;&7 6 &1. NL Add &) &W &p &4 NL &i.;7;1;1;1;2;1;&7 6 &1. NL Add 1 #yShiv &p &4 NL &i&5Protect;0;-;-1;-1;+;0;14,7;&D. NL &7 #g16 &1.;14,7;2;2;1;2;3;&D. NL &7 12 &1&5Go &^ &r Eyes;0;-;-1;-1;+;0;15;&3 &: &0. NL If &r &B intends to NL &I, &[ &) &x.;15;0;0;0;2;2;&3 3 &0. NL If &r &B intends to NL &I, &[ 1 &x&5Masterful Stab;0;-;-1;-1;+;0;4;&` 1 &g &X NL &^ &M time &A &- NL HP &f &k. NL &3 #g16 &0.;4;0;0;0;3;1;&` 1 &g &X NL &^ &M time &A &- NL HP &f &k. NL &3 12 &0&5Empty Fist;0;-;-1;-1;+;0;16;&3 #g14 &0. NL Exit &4 &S.;16;1;1;0;2;3;&3 9 &0. NL Exit &4 &S&5Backflip;0;-;-1;-1;+;0;7;&7 &# &1. NL &m 2 &6s.;7;1;1;1;2;1;&7 5 &1. NL &m 2 &6s&5Third Eye;0;-;-1;-1;+;0;7,13;&7 #g9 &1. NL &P !M!.;7,13;1;1;1;2;3;&7 7 &1. NL &P !M!&5Recycle;0;-;-1;-1;+;0;11,4;&2 a &6. NL &7 &X &] to its NL cost.;11,4;1;0;1;3;2;&2 a &6. NL &7 &X &] to its NL cost&5&G &F;0;-;-1;-1;+;0;11;&G &4 &F. NL &2.;11;1;0;1;3;2;&G &4 &F. NL &2&5Headbutt;0;-;-1;-1;+;0;-;&3 #g12 &0. NL &( a &6 &s &4 NL dis&6 &j on top of NL &4 &w &j.;-;1;1;0;2;0;&3 9 &0. NL &( a &6 &s &4 NL dis&6 &j on top of NL &4 &w &j&5Uppercut;0;-;-1;-1;+;0;15,6;&3 13 &0. NL &d &) &x. NL &d &) &9.;15,6;2;2;0;3;0;&3 13 &0. NL &d 1 &x. NL &d 1 &9&5Stack;0;-;-1;-1;+;0;7;&7 &1 &] to &r NL &Q of &6s in NL &4 dis&6 &j +3.;7;1;1;1;2;2;&7 &1 &] to &r NL &Q of &6s in NL &4 dis&6 &j&5Deadly &@;0;-;-1;-1;+;0;12;&d #g7 &a.;12;1;1;1;2;1;&d 5 &a&5Shrug It Off;0;-;-1;-1;+;0;7;&7 #g11 &1. NL &m 1 &6.;7;1;1;1;2;0;&7 8 &1. NL &m 1 &6&5Clash;0;-;-1;-1;+;0;-;Can &~ be &q if NL every &6 in &4 NL &i is an &z. NL &3 #g18 &0.;-;0;0;0;2;0;Can &~ be &q if NL every &6 in &4 NL &i is an &z. NL &3 14 &0&5Flying Sleeves;0;-;-1;-1;+;0;14;&D. NL &3 #g6 &0 &L.;14;1;1;0;2;3;&D. NL &3 4 &0 &L&5Follow-Up;0;-;-1;-1;+;0;4;&3 #g11 &0. NL If &r last &6 &q NL &f &k was an NL &z, &n &X.;4;1;1;0;2;3;&3 7 &0. NL If &r last &6 &q NL &f &k was an NL &z, &n &X&5Acrobatics;0;-;-1;-1;+;0;-;&m &: &6s. NL Dis&6 1 &6.;-;1;1;1;2;1;&m 3 &6s. NL Dis&6 1 &6&5Anger;0;-;-1;-1;+;0;-;&3 &# &0. NL Add a &} of &f &6 NL &p &4 dis&6 &j.;-;0;0;0;2;0;&3 6 &0. NL Add a &} of &f &6 NL &p &4 dis&6 &j&5Iron Wave;0;-;-1;-1;+;0;7;&7 #g7 &1. NL &3 #g7 &0.;7;1;1;0;2;0;&7 5 &1. NL &3 5 &0&5Hologram;0;-;-1;-1;+;0;7;&7 #g5 &1. NL &( a &6 &s &4 NL dis&6 &j &p &4 NL &i.;7,11;1;1;1;2;2;&7 3 &1. NL &( a &6 &s &4 NL dis&6 &j &p &4 NL &i. NL &2&5Deflect;0;-;-1;-1;+;0;7;&7 #g7 &1.;7;0;0;1;2;1;&7 4 &1&5Sword Boomerang;0;-;-1;-1;+;0;-;&3 3 &0 to a NL &o &B &: NL &v.;-;1;1;0;2;0;&3 3 &0 to a NL &o &B 3 NL &v&5Crippling Cloud;0;-;-1;-1;+;0;12,15,11;&d #g7 &a &' 2 NL &x to &% &h. NL &2.;12,15,11;2;2;1;3;1;&d 4 &a &' 2 NL &x to &% &h. NL &2&5Violence;0;-;-1;-1;+;0;11;&( &: &o &zs NL &s &4 &w &j NL &p &4 &i. NL &2.;11;0;0;1;4;4;&( 3 &o &zs NL &s &4 &w &j NL &p &4 &i. NL &2&5H&' of Greed;0;-;-1;-1;+;0;17;&3 &)5 &0. NL If #yFatal, &n &)5 Gold.;17;2;2;0;4;4;&3 20 &0. NL If #yFatal, &n 20 Gold&5Swift &y;0;-;-1;-1;+;0;-;&3 &! &0.;-;0;0;0;3;4;&3 7 &0&5Apo&rosis;0;-;-1;-1;+;0;18,11;&; &% &4 NL &6s &^ &r rest of NL &k. NL &2.;18,11;2;1;1;4;4;&; &% &4 NL &6s &^ &r rest of NL &k. NL &2&5Master of Strategy;0;-;-1;-1;+;0;11;&m &: &6s. NL &2.;11;0;0;1;4;4;&m 3 &6s. NL &2&5Searing Blow+2000;0;-;-1;-1;Searing Blow+2001;2000;_;&3 &)009016 &0. NL Can be &+ any NL &Q of &v.;18;2;2;0;3;0;&3 2007012 &0. NL Can be &+ any NL &Q of &v&5Shiv;0;-;-1;-1;+;0;11;&3 #g6 &0. NL &2.;11;0;0;0;1;4;&3 4 &0. NL &2&5;&{;&{ adds &g &0 to &Is&5Intangible;Reduce &% &0 taken &' HP loss to #b1&5&.;&. improves Block &ned &s &6s&5E&rreal;If &f &6 is in &4 &i at &r end of &e, it is exhausted. Exhausted &6s are removed &s &4 deck until &r end of &k&5&X &F;&F is used to &R &6s &s &4 &i&5Un&Rable;Un&Rable &6s cannot be &q &s &4 &i&5&*;&* &O take #b50% more &0 &s &zs&5Block;Until &< &e, prevents &0&5Channel;Channeling an Orb puts it &p &4 &/ empty slot. If &A &V no empty slots, &4 &/ Orb is automatically #yEvoked to make room&5Lightning;Orb: &3s &0 to &o &h&5Wrath;In &f &S, &A deal &' &Y double &I &0&5Exhaust;Removed until end of &k&5&@;&@ed &O &- HP at &r &C of &rir &e. Each &e, &@ is reduced by #b1&5Scry;Look at &r top X &6s of &4 &w &j. You may dis&6 any of &rm&5Retain;Not dis&6ed at &r end of &4 &e&5Weak;Weakened &O deal #b25% less &0 with &zs&5Stance;You can &~ &V one stance at a time&5Fatal;Triggers whenever &f &6 kills a non-minion &B&5Upgrade;Upgrading &6s makes &rm more powerful. Cards can &~ be upgraded once."

func TestDebug(t *testing.T) {
	api := &API{
		deckLists: map[string]string{
			"test": megaDeckNew,
		},
		deckLock: &sync.RWMutex{},
	}
	api.getDeckHandler(nil)
}
