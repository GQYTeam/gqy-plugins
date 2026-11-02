'use strict';

const grid = document.getElementById('grid');
const count = document.getElementById('count');
const message = document.getElementById('message');
const search = document.getElementById('search');
const updated = document.getElementById('updated');

let plugins = [];

function escapeHTML(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function compareVersion(a, b) {
  const pa = String(a).split('.').map((x) => Number(x) || 0);
  const pb = String(b).split('.').map((x) => Number(x) || 0);
  for (let i = 0; i < 3; i += 1) {
    if (pa[i] !== pb[i]) return (pa[i] || 0) - (pb[i] || 0);
  }
  return 0;
}

function latestVersion(plugin) {
  const versions = (plugin.versions || []).filter((v) => v.yanked === false);
  versions.sort((a, b) => compareVersion(b.version, a.version));
  return versions[0] || null;
}

function formatSize(bytes) {
  if (typeof bytes !== 'number' || bytes <= 0) return '大小未知';
  const units = ['B', 'KiB', 'MiB', 'GiB'];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i += 1;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function installCommand(plugin, version) {
  return `go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest install ${plugin.id}@${version}`;
}

function cardHTML(plugin) {
  const version = latestVersion(plugin);
  const tags = (plugin.tags || []).map((tag) => `<span class="tag">${escapeHTML(tag)}</span>`).join('');
  const actions = version && Array.isArray(version.actions) && version.actions.length > 0
    ? version.actions.map((name) => `<code>${escapeHTML(name)}</code>`).join(' · ')
    : '未声明动作';
  const command = version ? installCommand(plugin, version.version) : '目录暂无可用版本';
  return `
    <article class="card">
      <div class="card-head">
        <h2>${escapeHTML(plugin.name)}</h2>
        <span class="badge">${escapeHTML(version ? version.version : '无版本')}</span>
      </div>
      <p>${escapeHTML(plugin.summary)}</p>
      <div class="tags">${tags}</div>
      <div class="actions">动作：${actions}</div>
      <div class="install-row">
        <code title="${escapeHTML(command)}">${escapeHTML(command)}</code>
        <button class="copy" type="button" data-command="${escapeHTML(command)}">复制</button>
      </div>
      <div class="meta">${escapeHTML(plugin.id)} · ${escapeHTML(version ? version.protocol || '' : '')} · ${formatSize(version ? version.size : 0)}</div>
    </article>`;
}

function render() {
  const query = search.value.trim().toLowerCase();
  const visible = plugins.filter((plugin) => {
    if (query === '') return true;
    const haystack = [plugin.id, plugin.name, plugin.summary, plugin.description, ...(plugin.tags || [])]
      .join(' ')
      .toLowerCase();
    return haystack.includes(query);
  });
  if (visible.length === 0) {
    grid.innerHTML = '';
    message.className = query === '' ? 'message' : 'message error';
    message.textContent = query === '' ? '目录暂时没有插件。' : `没有匹配「${query}」的插件。`;
  } else {
    grid.innerHTML = visible.map(cardHTML).join('');
    message.textContent = '';
    message.className = 'message';
  }
  count.textContent = `${visible.length} / ${plugins.length} 个插件`;
}

async function copyText(text, button) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const area = document.createElement('textarea');
    area.value = text;
    document.body.append(area);
    area.select();
    document.execCommand('copy');
    area.remove();
  }
  const previous = button.textContent;
  button.textContent = '已复制';
  window.setTimeout(() => { button.textContent = previous; }, 1200);
}

async function loadRegistry() {
  try {
    const response = await fetch('./registry.json', { cache: 'no-store' });
    if (response.ok === false) {
      throw new Error(`registry.json 返回 HTTP ${response.status}`);
    }
    const doc = await response.json();
    if (Array.isArray(doc.plugins) === false) {
      throw new Error('registry.json 缺少 plugins 数组');
    }
    plugins = doc.plugins;
    updated.textContent = doc.updated_at ? `目录更新于 ${new Date(doc.updated_at).toLocaleString('zh-CN')}` : '';
    render();
  } catch (error) {
    count.textContent = '';
    message.className = 'message error';
    message.textContent = `读取目录失败：${error.message}`;
  }
}

search.addEventListener('input', render);
document.addEventListener('click', (event) => {
  const button = event.target.closest('button[data-command], button[data-copy]');
  if (button === null) return;
  const text = button.dataset.command || button.dataset.copy || '';
  if (text === '') return;
  copyText(text, button);
});
loadRegistry();
