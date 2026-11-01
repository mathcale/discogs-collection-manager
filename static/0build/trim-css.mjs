import fs from 'node:fs';
import postcss from 'postcss';

const src = process.argv[2];
const out = process.argv[3];
const usedClasses = new Set(
  fs.readFileSync(process.argv[4], 'utf8').split('\n').filter(Boolean),
);
const components = ['alert', 'button', 'card', 'drop', 'form', 'modal', 'nav', 'notification', 'webc', 'static'];
const keepBy = {
  webc: s => /\.z-(input-pin|custom-select)\b/.test(s),
  static: s =>
    /\.z-(spinner|transition|animation)/.test(s) ||
    (/\.z-(position|overflow)/.test(s) && usesOnlyKnownClasses(s)),
};
const prunable = new Set(['alert', 'button', 'card', 'drop', 'form', 'modal', 'nav', 'notification']);
const classesIn = selector => [...selector.matchAll(/\.(z-[a-zA-Z0-9_-]+)/g)].map(m => m[1]);
const runtimeBuiltPrefixes = ['z-notification-message-', 'z-notification-top-', 'z-notification-bottom-'];
const isKnownClass = c => usedClasses.has(c) || runtimeBuiltPrefixes.some(p => c.startsWith(p));
const usesOnlyKnownClasses = selector => classesIn(selector).every(isKnownClass);
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
  if (prunable.has(c)) f = filterLayer(f, usesOnlyKnownClasses);
  css += f + '\n';
}
fs.writeFileSync(out, css);
