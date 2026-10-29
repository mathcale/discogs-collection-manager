import fs from 'node:fs';
import postcss from 'postcss';

const src = process.argv[2];
const out = process.argv[3];
const components = ['alert', 'button', 'card', 'form', 'modal', 'nav', 'notification', 'webc', 'static'];
const keepBy = {
  webc: s => /\.z-input-pin\b/.test(s),
  static: s =>
    /\.z-(spinner|transition|animation|position|overflow)/.test(s),
};
const read = f => fs.readFileSync(`${src}/src/css/${f}`, 'utf8');

function filterLayer(css, keep) {
  const root = postcss.parse(css);
  root.walkRules(rule => {
    if (rule.parent.type === 'atrule' && rule.parent.name === 'keyframes') return;
    const kept = rule.selectors.filter(keep);
    if (kept.length === 0) rule.remove();
    else rule.selectors = kept;
  });
  root.walkAtRules(a => {
    if (a.nodes && a.nodes.length === 0) a.remove();
  });
  return root.toString();
}

let css = '@layer theme, base, components, styles, utilities;\n';
const core = read('core.css');
css += core.slice(core.indexOf('@layer base')) + '\n';
css += read('kit-core.css') + '\n';
for (const c of components) {
  let f = read(`kit-components/${c}.css`);
  if (keepBy[c]) f = filterLayer(f, keepBy[c]);
  css += f + '\n';
}
fs.writeFileSync(out, css);
