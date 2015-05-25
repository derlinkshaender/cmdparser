# cmdparser
Simple command line parser for shell-based tools in GoLang

I'm currently learning Go, so I decided to do a small parser project that 
allows me to define grammars for simple command-line utilities.

This is a work in progress and the code will change considerably every time I learn something new in Go.

## Grammar Structure

Right now you specify the grammar using something that resembles a PEG (parser expression grammar).

  	Grammar := map[string]string{
  		"START":         `"show"  FeatureClause     Options  ToClause? `,
  		"ToClause":      `"to"  !string `,
  		"FeatureClause": `"feature"  !string? `,
  		"Options":       `TranClause | DefClause | ValueClause `,
  		"TranClause":    `"translation"  LangList? `,
  		"LangList":      `"lang"  !string? `,
  		"ValueClause":   `"unique"?  "values" `,
  		"DefClause":     `"definition" `,
  	}

