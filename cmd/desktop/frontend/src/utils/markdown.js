/**
 * 简易 Markdown 解析器
 * 支持常用格式：标题、粗体、斜体、删除线、行内代码、代码块、链接、图片、列表、引用、分割线
 */

/**
 * 转义 HTML 特殊字符，防止 XSS
 * @param {string} text
 * @returns {string}
 */
function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, char => map[char]);
}

/**
 * 解析 Markdown 表格
 * @param {string} text
 * @returns {string}
 */
function parseTable(text) {
    const lines = text.split('\n');
    const result = [];
    let inTable = false;
    let tableRows = [];

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();

        // 检测表格行：以 | 开头或包含 | 分隔的内容
        if (line.match(/^\|(.+)\|$/)) {
            // 检查是否是分隔行 |---|---|
            if (line.match(/^\|[\s:-]+\|$/)) {
                // 分隔行，标记表格开始
                if (tableRows.length > 0) {
                    inTable = true;
                }
                continue;
            }

            // 解析表格单元格
            const cells = line
                .slice(1, -1) // 去掉首尾的 |
                .split('|')
                .map(cell => cell.trim());

            tableRows.push(cells);
        } else {
            // 非表格行，输出之前收集的表格
            if (tableRows.length > 0) {
                result.push(buildTable(tableRows, inTable));
                tableRows = [];
                inTable = false;
            }
            result.push(line);
        }
    }

    // 处理末尾的表格
    if (tableRows.length > 0) {
        result.push(buildTable(tableRows, inTable));
    }

    return result.join('\n');
}

/**
 * 构建 HTML 表格
 * @param {Array} rows - 表格行数据
 * @param {boolean} hasHeader - 是否有表头
 * @returns {string}
 */
function buildTable(rows, hasHeader) {
    if (rows.length === 0) return '';

    let html = '<table class="md-table">';

    if (hasHeader && rows.length > 0) {
        // 第一行作为表头
        html += '<thead><tr>';
        rows[0].forEach(cell => {
            html += `<th>${cell}</th>`;
        });
        html += '</tr></thead>';

        // 剩余行作为表体
        if (rows.length > 1) {
            html += '<tbody>';
            for (let i = 1; i < rows.length; i++) {
                html += '<tr>';
                rows[i].forEach(cell => {
                    html += `<td>${cell}</td>`;
                });
                html += '</tr>';
            }
            html += '</tbody>';
        }
    } else {
        // 没有表头，全部作为表体
        html += '<tbody>';
        rows.forEach(row => {
            html += '<tr>';
            row.forEach(cell => {
                html += `<td>${cell}</td>`;
            });
            html += '</tr>';
        });
        html += '</tbody>';
    }

    html += '</table>';
    return html;
}

/**
 * 解析 Markdown 文本为 HTML
 * @param {string} text - Markdown 文本
 * @returns {string} - HTML 字符串
 */
export function parseMarkdown(text) {
    if (!text) return '';

    // 存储代码块，避免被其他规则处理
    const codeBlocks = [];
    const inlineCodes = [];

    // 1. 先提取代码块 ```code```
    text = text.replace(/```(\w*)\n?([\s\S]*?)```/g, (match, lang, code) => {
        const index = codeBlocks.length;
        codeBlocks.push({ lang, code: escapeHtml(code.trim()) });
        return `\x00CODEBLOCK${index}\x00`;
    });

    // 2. 提取行内代码 `code`
    text = text.replace(/`([^`]+)`/g, (match, code) => {
        const index = inlineCodes.length;
        inlineCodes.push(escapeHtml(code));
        return `\x00INLINECODE${index}\x00`;
    });

    // 3. 转义 HTML（代码已经单独处理了）
    text = escapeHtml(text);

    // 4. 标题 # ## ### #### ##### ######
    text = text.replace(/^######\s+(.+)$/gm, '<h6>$1</h6>');
    text = text.replace(/^#####\s+(.+)$/gm, '<h5>$1</h5>');
    text = text.replace(/^####\s+(.+)$/gm, '<h4>$1</h4>');
    text = text.replace(/^###\s+(.+)$/gm, '<h3>$1</h3>');
    text = text.replace(/^##\s+(.+)$/gm, '<h2>$1</h2>');
    text = text.replace(/^#\s+(.+)$/gm, '<h1>$1</h1>');

    // 5. 分割线 --- *** ___
    text = text.replace(/^([-*_]){3,}\s*$/gm, '<hr>');

    // 6. 粗体 **text** 或 __text__
    text = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    text = text.replace(/__(.+?)__/g, '<strong>$1</strong>');

    // 7. 斜体 *text* 或 _text_（注意不要匹配已处理的粗体）
    text = text.replace(/\*([^*]+)\*/g, '<em>$1</em>');
    text = text.replace(/(?<![a-zA-Z0-9])_([^_]+)_(?![a-zA-Z0-9])/g, '<em>$1</em>');

    // 8. 删除线 ~~text~~
    text = text.replace(/~~(.+?)~~/g, '<del>$1</del>');

    // 9. 图片 ![alt](url)
    text = text.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width: 100%;">');

    // 10. 链接 [text](url)
    text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');

    // 11. 引用块 > text
    text = text.replace(/^&gt;\s+(.+)$/gm, '<blockquote>$1</blockquote>');
    // 合并连续的引用块
    text = text.replace(/<\/blockquote>\n<blockquote>/g, '\n');

    // 12. 无序列表 - item 或 * item
    text = text.replace(/^[-*]\s+(.+)$/gm, '<li>$1</li>');
    text = text.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

    // 13. 有序列表 1. item
    text = text.replace(/^\d+\.\s+(.+)$/gm, '<oli>$1</oli>');
    text = text.replace(/(<oli>.*<\/oli>\n?)+/g, match => {
        return '<ol>' + match.replace(/<\/?oli>/g, tag => tag === '<oli>' ? '<li>' : '</li>') + '</ol>';
    });

    // 14. 表格解析
    text = parseTable(text);

    // 15. 换行处理
    text = text.replace(/\n/g, '<br>');

    // 16. 清理多余的 <br>
    text = text.replace(/<br><(h[1-6]|ul|ol|blockquote|hr|table)/g, '<$1');
    text = text.replace(/<\/(h[1-6]|ul|ol|blockquote|table)><br>/g, '</$1>');
    text = text.replace(/<hr><br>/g, '<hr>');

    // 16. 还原代码块
    text = text.replace(/\x00CODEBLOCK(\d+)\x00/g, (match, index) => {
        const { lang, code } = codeBlocks[index];
        const langClass = lang ? ` class="language-${lang}"` : '';
        return `<pre><code${langClass}>${code}</code></pre>`;
    });

    // 17. 还原行内代码
    text = text.replace(/\x00INLINECODE(\d+)\x00/g, (match, index) => {
        return `<code class="inline-code">${inlineCodes[index]}</code>`;
    });

    return text;
}
