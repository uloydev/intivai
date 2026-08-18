import { memo } from "react"
import ReactMarkdown, { type Components } from "react-markdown"
import remarkGfm from "remark-gfm"
import { cn } from "@/lib/utils"

const markdownComponents: Components = {
  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
  a: ({ children, href }) => (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="text-primary underline underline-offset-2 hover:opacity-80"
    >
      {children}
    </a>
  ),
  strong: ({ children }) => <strong className="font-semibold text-foreground">{children}</strong>,
  ul: ({ children }) => <ul className="mb-2 list-disc pl-5 last:mb-0 space-y-1">{children}</ul>,
  ol: ({ children }) => <ol className="mb-2 list-decimal pl-5 last:mb-0 space-y-1">{children}</ol>,
  li: ({ children }) => <li className="leading-relaxed">{children}</li>,
  h1: ({ children }) => <h1 className="mb-2 mt-3 text-base font-bold first:mt-0">{children}</h1>,
  h2: ({ children }) => <h2 className="mb-2 mt-3 text-sm font-bold first:mt-0">{children}</h2>,
  h3: ({ children }) => <h3 className="mb-1.5 mt-2.5 text-sm font-semibold first:mt-0">{children}</h3>,
  blockquote: ({ children }) => (
    <blockquote className="mb-2 border-l-2 border-border pl-3 text-muted-foreground last:mb-0">
      {children}
    </blockquote>
  ),
  code: ({ className, children }) => {
    const isBlock = /language-|^lang-/.test(className ?? "") || String(children).includes("\n")
    if (isBlock) {
      return (
        <pre className="mb-2 overflow-x-auto rounded-lg border border-border/60 bg-muted/60 p-3 text-[11px] leading-relaxed last:mb-0">
          <code className={className}>{children}</code>
        </pre>
      )
    }
    return (
      <code className="rounded bg-muted/60 px-1 py-0.5 font-mono text-[0.92em] text-foreground">
        {children}
      </code>
    )
  },
  table: ({ children }) => (
    <div className="mb-2 overflow-x-auto last:mb-0">
      <table className="w-full border-collapse text-left text-xs">{children}</table>
    </div>
  ),
  th: ({ children }) => (
    <th className="border border-border/60 bg-muted/40 px-2 py-1.5 font-semibold">{children}</th>
  ),
  td: ({ children }) => <td className="border border-border/60 px-2 py-1.5">{children}</td>,
}

export interface MarkdownProps {
  content: string
  className?: string
}

export const Markdown = memo(function Markdown({ content, className }: MarkdownProps) {
  return (
    <div className={cn("text-xs sm:text-sm leading-relaxed", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {content}
      </ReactMarkdown>
    </div>
  )
})
