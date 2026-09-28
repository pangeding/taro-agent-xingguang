/**
 * 轻量 Markdown 渲染。
 *
 * 原为 `app/chat/page.jsx` 内的 SimpleMarkdown，占卜页也需要渲染 AI 解读，
 * 故抽为共享组件。行为与抽取前完全一致：支持围栏代码块、行内 code、
 * 粗体、斜体、# ~ ### 标题。不支持的语法按纯文本输出。
 */
export default function Markdown({ content }) {
  if (!content) return null

  const parts = content.split(/(```[\s\S]*?```|`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|#{1,3} .+?)(?:\n|$)/g)

  return (
    <div className="markdown-content">
      {parts.map((part, i) => {
        if (part.startsWith('```')) {
          const code = part.replace(/^```\w*\n?|\n?```$/g, '')
          return (
            <pre key={i} className="bg-mystic-800 text-mystic-100 p-3 rounded-lg my-2 overflow-x-auto text-sm">
              <code>{code}</code>
            </pre>
          )
        }
        if (part.startsWith('`') && part.endsWith('`')) {
          return <code key={i} className="bg-mystic-100 text-mystic-800 px-1 py-0.5 rounded text-sm">{part.slice(1, -1)}</code>
        }
        if (part.startsWith('**') && part.endsWith('**')) {
          return <strong key={i} className="font-bold">{part.slice(2, -2)}</strong>
        }
        if (part.startsWith('*') && part.endsWith('*') && !part.startsWith('**')) {
          return <em key={i}>{part.slice(1, -1)}</em>
        }
        if (part.startsWith('### ')) {
          return <h3 key={i} className="text-lg font-bold mt-3 mb-1">{part.slice(4)}</h3>
        }
        if (part.startsWith('## ')) {
          return <h2 key={i} className="text-xl font-bold mt-3 mb-2">{part.slice(3)}</h2>
        }
        if (part.startsWith('# ')) {
          return <h1 key={i} className="text-2xl font-bold mt-3 mb-2">{part.slice(2)}</h1>
        }
        return <span key={i}>{part}</span>
      })}
    </div>
  )
}
