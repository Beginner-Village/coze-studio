// Use require() to resolve plugins from postcss-config's node_modules
// to work around pnpm phantom dependency issues
const path = require('path');
const configPkgDir = path.dirname(require.resolve('@coze-arch/postcss-config/package.json'));
const configNodeModules = path.join(configPkgDir, 'node_modules');

function loadPlugin(name) {
  try {
    return require(name);
  } catch {
    return require(path.join(configNodeModules, name));
  }
}

module.exports = {
  plugins: [
    loadPlugin('postcss-import'),
    loadPlugin('tailwindcss/nesting')(loadPlugin('postcss-nesting')),
    require('tailwindcss'),
    loadPlugin('@csstools/postcss-is-pseudo-class'),
  ],
};
