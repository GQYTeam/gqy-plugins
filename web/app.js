'use strict';

const grid = document.getElementById('grid');
const count = document.getElementById('count');
const message = document.getElementById('message');
const search = document.getElementById('search');
const updated = document.getElementById('updated');
const heroCommandEl = document.getElementById('hero-command');

const CLI_INSTALL = 'go install github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest';
const GQY_SERVER = 'http://127.0.0.1:5173';
const HERO_COMMAND = `${CLI_INSTALL}\n"$(go env GOPATH)/bin/gqy-plugin" install <id> --server ${GQY_SERVER} --load`;

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
  const versions = (plugin.versions || []).filter((v) => v.yanked !== true);
  versions.sort((a, b) => compareVersion(b.version, a.version));
  return versions[0] || null;
}

function formatSize(bytes) {
  if (typeof bytes !== 'number' || bytes <= 0) return 'SIZE UNKNOWN';
  const units = ['B', 'KiB', 'MiB', 'GiB'];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i += 1;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function installCommand(id, version) {
  return `${CLI_INSTALL} && "$(go env GOPATH)/bin/gqy-plugin" install ${id}@${version} --server ${GQY_SERVER} --load`;
}

function iconSVG(id) {
  if (id.includes('image')) {
    return '<svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"></rect><circle cx="8.5" cy="9" r="1.5"></circle><path d="m4 17 5-5 4 4 2-2 5 4"></path></svg>';
  }
  if (id.includes('emote')) {
    return '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"></circle><path d="M8 14.5c1 1.2 2.3 1.8 4 1.8s3-.6 4-1.8"></path><path d="M9 9.5h.01M15 9.5h.01"></path></svg>';
  }
  if (id.includes('github')) {
    return '<svg viewBox="0 0 24 24"><circle cx="8" cy="15" r="4"></circle><path d="M11 12 20 3M17 6l3 3M5 3l4 4M4 8l4-4"></path></svg>';
  }
  if (id.includes('migrat')) {
    return '<svg viewBox="0 0 24 24"><path d="M4 7h15M15 3l4 4-4 4M20 17H5M9 13l-4 4 4 4"></path></svg>';
  }
  return '<svg viewBox="0 0 24 24"><path d="M12 3 4 7v10l8 4 8-4V7l-8-4Z"></path><path d="m4 7 8 4 8-4M12 11v10"></path></svg>';
}

function repoLabel(plugin) {
  return String(plugin.repository || plugin.homepage || '').replace(/^https?:\/\/github\.com\//, '');
}

function sourceBadge(plugin) {
  if (plugin.source_type === 'external') return '<span class="source external">EXTERNAL · PR</span>';
  if (plugin.source_type === 'org') return '<span class="source org">ORG · AUTO</span>';
  return '<span class="source">DIRECTORY</span>';
}

function cardHTML(plugin) {
  const version = latestVersion(plugin);
  const mode = version ? (version.mode || 'resident') : 'resident';
  const oneshot = mode === 'oneshot';
  const versionLabel = version ? `v${version.version}` : 'NO VERSION';
  const tags = (plugin.tags || []).slice(0, 4).map((tag) => `<span class="tag">${escapeHTML(tag)}</span>`).join('');
  const actions = version && Array.isArray(version.actions) ? version.actions : [];
  const actionItems = actions.slice(0, 4).map((name) => `<code>${escapeHTML(name)}</code>`).join('<span aria-hidden="true"> · </span>');
  const extraActions = actions.length > 4 ? `<span> +${actions.length - 4}</span>` : '';
  const command = version ? installCommand(plugin.id, version.version) : '';
  const actionMarkup = actions.length > 0 ? `<div class="actions">ACTIONS<span class="action-list">${actionItems}${extraActions}</span></div>` : '';
  let installMarkup = '';
  let commandMarkup = '';
  if (version && oneshot === false) {
    installMarkup = `<button class="install-btn" type="button" data-install="${escapeHTML(plugin.id)}" data-version="${escapeHTML(version.version)}" title="复制：先安装 CLI，再安装插件并自动加载"><span>安装并加载</span><span class="arrow">↗</span></button>`;
    commandMarkup = `<details class="command-box"><summary>查看安装命令</summary><pre>${escapeHTML(command)}</pre></details>`;
  } else if (version && oneshot) {
    installMarkup = '<button class="install-btn disabled" type="button" disabled><span>一次性插件</span><span class="arrow">—</span></button>';
    commandMarkup = '<details class="command-box"><summary>运行方式</summary><pre>oneshot 插件不常驻安装。请使用 GQY 的一次性插件运行入口。</pre></details>';
  } else {
    installMarkup = '<button class="install-btn disabled" type="button" disabled><span>暂无可用版本</span><span class="arrow">—</span></button>';
  }
  return `
    <article class="card">
      <div class="card-top">
        <div class="plugin-icon" aria-hidden="true">${iconSVG(plugin.id)}</div>
        <div class="badges">
          ${sourceBadge(plugin)}
          ${oneshot ? '<span class="source oneshot">ONESHOT</span>' : ''}
          <span class="badge">${escapeHTML(versionLabel)}</span>
        </div>
      </div>
      <h3 class="card-title">${escapeHTML(plugin.name)}</h3>
      <p class="repo">${escapeHTML(repoLabel(plugin) || plugin.id)}</p>
      <p class="summary">${escapeHTML(plugin.summary)}</p>
      <div class="tags">${tags}</div>
      ${actionMarkup}
      <div class="card-foot">
        <div class="install-row">${installMarkup}</div>
        ${commandMarkup}
        <div class="card-meta">
          <span>${escapeHTML(plugin.id)}</span>
          <span>${escapeHTML(version ? version.protocol || 'unknown' : 'unknown')} · ${formatSize(version ? version.size : 0)}</span>
        </div>
      </div>
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
  count.textContent = `${String(visible.length).padStart(2, '0')} / ${String(plugins.length).padStart(2, '0')} PLUGINS`;
  if (visible.length === 0) {
    grid.innerHTML = '';
    message.className = query === '' ? 'message' : 'message error';
    message.textContent = query === '' ? '目录暂时没有插件。' : `没有匹配「${query}」的插件。`;
    return;
  }
  grid.innerHTML = visible.map(cardHTML).join('');
  message.textContent = '';
  message.className = 'message';
}

async function copyText(text, button, idleLabel) {
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
  if (button === null) return;
  const previous = idleLabel || button.textContent;
  button.textContent = '已复制 ✓';
  window.setTimeout(() => { button.textContent = previous; }, 1400);
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
    updated.textContent = doc.updated_at ? `UPDATED ${new Date(doc.updated_at).toLocaleString('zh-CN')}` : '';
    render();
  } catch (error) {
    count.textContent = '00 / 00 PLUGINS';
    message.className = 'message error';
    message.textContent = `读取目录失败：${error.message}`;
  }
}

search.addEventListener('input', render);
document.addEventListener('click', (event) => {
  const installButton = event.target.closest('button[data-install]');
  if (installButton !== null) {
    const command = installCommand(installButton.dataset.install, installButton.dataset.version);
    copyText(command, installButton.querySelector('span') || installButton, '安装并加载');
    return;
  }
  const heroButton = event.target.closest('button[data-hero-copy]');
  if (heroButton !== null) {
    copyText(HERO_COMMAND, heroButton, '复制');
  }
});
document.addEventListener('keydown', (event) => {
  if (event.key === '/' && document.activeElement !== search) {
    event.preventDefault();
    search.focus();
  }
});
if (heroCommandEl !== null) {
  heroCommandEl.setAttribute('title', HERO_COMMAND);
}
loadRegistry();
