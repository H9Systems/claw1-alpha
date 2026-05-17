import React from 'react'
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

function renderInline(text: string) {
  const parts = text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g)
  return parts.map((part, index) => {
    if (part.startsWith('`') && part.endsWith('`')) {
      return (
        <code
          key={index}
          className="inline-block px-1.5 py-0.5 border border-code-border rounded bg-code-bg font-mono text-[0.82em]"
        >
          {part.slice(1, -1)}
        </code>
      )
    }
    if (part.startsWith('**') && part.endsWith('**')) {
      return (
        <strong key={index} className="text-accent font-bold">
          {part.slice(2, -2)}
        </strong>
      )
    }
    return <React.Fragment key={index}>{part}</React.Fragment>
  })
}

function SlideBody({ body }: { body: string[] }) {
  const blocks: React.ReactNode[] = []
  let list: string[] = []

  const flushList = () => {
    if (list.length === 0) return
    blocks.push(
      <ul key={`list-${blocks.length}`} className="grid gap-3 max-w-[880px] m-0 pl-[1.15em]">
        {list.map((item) => (
          <li key={item} className="max-w-[820px] m-0 text-[clamp(20px,2.5vw,34px)] leading-[1.24]">
            {renderInline(item.replace(/^-\s+/, ''))}
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
    if (line.trim() === '') {
      return
    }
    blocks.push(
      <p key={`p-${index}`} className="max-w-[820px] m-0 text-[clamp(20px,2.5vw,34px)] leading-[1.24]">
        {renderInline(line)}
      </p>,
    )
  })
  flushList()
  return <>{blocks}</>
}

function Deck() {
  return (
    <main className="w-[min(1180px,calc(100vw-40px))] mx-auto py-7 pb-14 max-md:w-[min(calc(100vw-24px),680px)] max-md:pt-2.5">
      {slides.map((slide, index) => (
        <section
          className="relative min-h-[82vh] grid content-center gap-[18px] px-[clamp(20px,5vw,72px)] py-16 border-b border-rule first:min-h-[92vh] max-md:min-h-auto max-md:px-1 max-md:py-[72px_4px_42px]"
          key={slide.title}
        >
          <div className="absolute top-7 right-0 text-[13px] text-ink-light tabular-nums">
            {String(index + 1).padStart(2, '0')}
          </div>
          <h1 className="max-w-[980px] m-0 text-[#111] text-[clamp(42px,7vw,92px)] leading-[0.95] tracking-normal">
            {slide.title}
          </h1>
          <SlideBody body={slide.body} />
        </section>
      ))}
    </main>
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
