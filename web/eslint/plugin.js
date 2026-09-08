// Understudy's own lint rules. One file per rule under rules/, each with a
// test beside it. Add a rule by adding a file and a line here.
import noPaletteClasses from './rules/no-palette-classes.js';
import noRawElements from './rules/no-raw-elements.js';

const plugin = {
	meta: { name: 'understudy' },
	rules: {
		'no-raw-elements': noRawElements,
		'no-palette-classes': noPaletteClasses,
	},
};

export default plugin;
