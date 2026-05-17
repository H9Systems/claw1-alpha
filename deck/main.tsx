import React, { useCallback, useEffect, useRef, useState } from 'react'
import ReactDOM from 'react-dom/client'
import {
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import pitch from '../PITCH.md?raw'
import './styles.css'

type Slide = {
  title: string
  body: string[]
}

const slides = parsePitch(pitch)

function parsePitch(markdown: string): Slide[] {
  const parsed: Slide[] = []
  let current: Slide | null = null

  for (const raw of markdown.split('\n')) {
    const line = raw.trimEnd()
    if (line.startsWith('## ')) {
      if (current) parsed.push(current)
      current = { title: line.replace(/^##\s+/, ''), body: [] }
      continue
    }
    if (line.startsWith('# ')) {
      if (current) parsed.push(current)
      current = { title: line.replace(/^#\s+/, ''), body: [] }
      continue
    }
    if (!current) {
      current = { title: 'Claw1', body: [] }
    }
    current.body.push(line)
  }
  if (current) parsed.push(current)
  return parsed.filter((slide) => slide.title || slide.body.some(Boolean))
}

/* ── Title renderer: "Claw" orange, "1" red ─────────────────────────────────── */

function SlideTitle({ text }: { text: string }) {
  if (text === 'Claw1') {
    return (
      <>
        <span className="text-claw-orange">Claw</span>
        <span className="text-claw-red">1</span>
      </>
    )
  }
  // Style any "Claw1" or "L1" occurrences within a longer title
  if (text.includes('Claw1') || text.includes('L1')) {
    const parts = text.split(/(Claw1|L1)/)
    const renderParts = () =>
      parts.map((part, i) => {
        if (part === 'Claw1') {
          return (
            <span key={i}>
              <span className="text-claw-orange">Claw</span>
              <span className="text-claw-red">1</span>
            </span>
          )
        }
        if (part === 'L1') {
          return (
            <span key={i} className="text-claw-red">L1</span>
          )
        }
        return <span key={i}>{part}</span>
      })
    // First slide: break after first word
    if (text.startsWith('Despliega')) {
      const [, afterDespliega] = text.split('Despliega ')
      const restParts = afterDespliega.split(/(Claw1|L1)/)
      const renderRest = () =>
        restParts.map((part, i) => {
          if (part === 'Claw1') {
            return (
              <span key={i} className="group/claw1">
                <span className="text-claw-orange group-hover:text-ink-muted group-hover/claw1:!text-claw-orange transition-colors">Claw</span>
                <span className="text-claw-red group-hover:text-ink-muted group-hover/claw1:!text-claw-red transition-colors">1</span>
              </span>
            )
          }
          if (part === 'L1') {
            return (
              <span key={i} className="text-claw-red">L1</span>
            )
          }
          return <span key={i} className="group-hover:text-ink-muted transition-colors">{part}</span>
        })
      return (
        <>
          <div className="text-claw-red group-hover:text-ink-muted transition-colors">DESPLIEG<span className="relative inline-flex items-center justify-center" style={{ width: '0.7em' }}><span className="opacity-0">A</span><svg className="absolute group-hover:fill-[#d42020] transition-colors" style={{ top: '0.17em', left: '0.08em', width: '0.7em', height: '0.7em' }} viewBox="1 1 14 14" fill="#d42020"><polygon points="8,1 15,15 1,15"/></svg></span> </div>
          <div className="mt-4 sm:mt-8">{renderRest()}</div>
        </>
      )
    }
    return <>{renderParts()}</>
  }
  return <>{text}</>
}

/* ── Copyable code block ────────────────────────────────────────────────────── */

function CodeBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false)
  const copy = () => {
    navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }
  return (
    <div className="group relative max-w-[880px]">
      <pre
        className="m-0 px-5 py-3.5 rounded-lg border border-code-border bg-code-bg font-mono text-code-text text-[clamp(14px,1.6vw,20px)] leading-relaxed overflow-x-auto"
      >
        <code>{code}</code>
      </pre>
      <button
        onClick={copy}
        className="absolute top-2 right-2 px-2 py-1 rounded text-[11px] font-mono border border-code-border bg-code-bg text-ink-muted opacity-0 group-hover:opacity-100 hover:text-accent hover:border-accent transition-all cursor-pointer"
      >
        {copied ? 'copiado' : 'copiar'}
      </button>
    </div>
  )
}

/* ── Inline renderer ────────────────────────────────────────────────────────── */

function renderInline(text: string) {
  const parts = text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g)
  return parts.map((part, index) => {
    if (part.startsWith('`') && part.endsWith('`')) {
      return (
        <code
          key={index}
          className="inline-block px-1.5 py-0.5 border border-code-border rounded bg-code-bg font-mono text-code-text text-[0.82em]"
        >
          {part.slice(1, -1)}
        </code>
      )
    }
    if (part.startsWith('**') && part.endsWith('**')) {
      return (
        <strong key={index} className="text-claw-orange font-bold">
          {part.slice(2, -2)}
        </strong>
      )
    }
    return <React.Fragment key={index}>{part}</React.Fragment>
  })
}

/* ── Slide body ─────────────────────────────────────────────────────────────── */

function SlideBody({ body }: { body: string[] }) {
  const blocks: React.ReactNode[] = []
  let list: string[] = []

  const flushList = () => {
    if (list.length === 0) return
    blocks.push(
      <ul key={`list-${blocks.length}`} className="grid gap-3 max-w-[880px] m-0 p-0 list-none">
        {list.map((item, i) => (
          <li key={i} className="flex items-start gap-2 sm:gap-3 max-w-[820px] m-0 text-[clamp(14px,3.5vw,28px)] leading-[1.35]">
            <span className="inline-block w-2 h-2 mt-[0.45em] shrink-0 bg-accent rounded-none" />
            <span>{renderInline(item.replace(/^-\s+/, ''))}</span>
          </li>
        ))}
      </ul>,
    )
    list = []
  }

  body.forEach((line, index) => {
    if (line.startsWith('- ')) {
      list.push(line)
      return
    }
    flushList()
    if (line.trim() === '') return

    // Standalone code line: entire line is a backtick-wrapped token
    const codeMatch = line.match(/^`([^`]+)`$/)
    if (codeMatch) {
      const inner = codeMatch[1]
      // If the content is a URL, render as a clickable link
      if (/^https?:\/\//.test(inner)) {
        blocks.push(
          <a
            key={`link-${index}`}
            href={inner}
            target="_blank"
            rel="noopener noreferrer"
            className="block max-w-[880px] px-5 py-3.5 rounded-lg border border-code-border bg-code-bg font-mono text-accent text-[clamp(14px,1.6vw,20px)] leading-relaxed no-underline hover:border-accent transition-colors"
          >
            {inner}
          </a>,
        )
        return
      }
      blocks.push(<CodeBlock key={`code-${index}`} code={inner} />)
      return
    }

    blocks.push(
      <p key={`p-${index}`} className="max-w-[820px] m-0 text-[clamp(14px,3.5vw,28px)] leading-[1.35]">
        {renderInline(line)}
      </p>,
    )
  })
  flushList()
  return <>{blocks}</>
}

/* ── Deck with horizontal arrow-key navigation ──────────────────────────────── */

function Deck() {
  const [current, setCurrent] = useState(0)
  const trackRef = useRef<HTMLDivElement>(null)
  const touchStart = useRef<{ x: number; y: number } | null>(null)

  const go = useCallback(
    (dir: 1 | -1) => {
      setCurrent((prev) => Math.max(0, Math.min(slides.length - 1, prev + dir)))
    },
    [],
  )

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
        e.preventDefault()
        go(1)
      }
      if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        e.preventDefault()
        go(-1)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [go])

  const handleTouchStart = (e: React.TouchEvent) => {
    const touch = e.touches[0]
    touchStart.current = { x: touch.clientX, y: touch.clientY }
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (!touchStart.current) return
    const touch = e.changedTouches[0]
    const dx = touch.clientX - touchStart.current.x
    const dy = touch.clientY - touchStart.current.y
    touchStart.current = null
    if (Math.abs(dx) > Math.abs(dy) && Math.abs(dx) > 40) {
      go(dx < 0 ? 1 : -1)
    }
  }

  return (
    <div className="fixed inset-0 overflow-hidden bg-bg">
      <a href="https://github.com/H9Systems/claw1-alpha" target="_blank" rel="noopener noreferrer" className="fixed top-3 left-4 sm:top-5 sm:left-8 z-10 flex items-baseline gap-2 sm:gap-3 select-none no-underline hover:opacity-80 transition-opacity">
        <span className="text-[22px] sm:text-[42px] font-extrabold tracking-tight">
          <span className="text-claw-green">H9</span><span className="text-ink-muted"> Systems</span>
        </span>
        <span className="text-[22px] sm:text-[42px] font-extrabold tracking-tight text-ink-muted">/</span>
        <span className="text-[22px] sm:text-[42px] font-extrabold tracking-tight">
          <span className="text-claw-orange">Claw</span><span className="text-claw-red">1</span>
        </span>
      </a>

      {/* Horizontal slide track */}
      <div
        ref={trackRef}
        className="flex h-full"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        style={{ transform: `translateX(-${current * 100}vw)`, transition: 'transform 0.45s cubic-bezier(0.4,0,0.2,1)' }}
      >
        {slides.map((slide, index) => (
          <section
            key={slide.title}
            className="shrink-0 w-screen h-full flex flex-col justify-center px-5 sm:px-[clamp(32px,8vw,120px)] py-16 sm:py-12 overflow-y-auto"
          >
            <div className={`max-w-[960px] mx-auto w-full flex flex-col gap-5${index === 0 ? ' items-center text-center' : ''}`}>
              <h1 className="m-0 text-accent text-[clamp(28px,7vw,80px)] sm:text-[clamp(36px,6vw,80px)] leading-[0.95] tracking-tight font-extrabold">
                {index === 0 ? (
                  <a href="https://github.com/H9Systems/claw1-alpha" target="_blank" rel="noopener noreferrer" className="group text-ink no-underline hover:text-ink-muted transition-colors">
                    <SlideTitle text={slide.title} />
                  </a>
                ) : (
                  <SlideTitle text={slide.title} />
                )}
              </h1>
              <SlideBody body={slide.body} />
            </div>
          </section>
        ))}
      </div>

      {/* Bottom bar: slide counter + dots + arrow hint */}
      <div className="fixed bottom-0 left-0 right-0 flex items-center justify-between px-4 sm:px-8 py-3 sm:py-4 text-ink-muted text-[11px] sm:text-[13px] select-none">
        <span className="tabular-nums font-mono">
          {String(current + 1).padStart(2, '0')}
          <span className="text-rule"> / </span>
          {String(slides.length).padStart(2, '0')}
        </span>

        {/* Dot indicators */}
        <div className="flex gap-1.5">
          {slides.map((_, i) => (
            <button
              key={i}
              onClick={() => setCurrent(i)}
              className={`w-2.5 h-2.5 rounded-none border-none cursor-pointer transition-all ${
                i === current
                  ? 'bg-accent scale-125'
                  : 'bg-rule hover:bg-ink-muted'
              }`}
            />
          ))}
        </div>

        <span className="font-mono text-rule hidden sm:inline">
          ← →
        </span>
      </div>
    </div>
  )
}

const rootRoute = createRootRoute({
  component: Deck,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: Deck,
})

const routeTree = rootRoute.addChildren([indexRoute])
const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>,
)
