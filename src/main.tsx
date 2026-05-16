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
      return <code key={index}>{part.slice(1, -1)}</code>
    }
    if (part.startsWith('**') && part.endsWith('**')) {
      return <strong key={index}>{part.slice(2, -2)}</strong>
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
      <ul key={`list-${blocks.length}`}>
        {list.map((item) => (
          <li key={item}>{renderInline(item.replace(/^-\s+/, ''))}</li>
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
    blocks.push(<p key={`p-${index}`}>{renderInline(line)}</p>)
  })
  flushList()
  return <>{blocks}</>
}

function Deck() {
  return (
    <main className="deck">
      {slides.map((slide, index) => (
        <section className="slide" key={slide.title}>
          <div className="slideIndex">{String(index + 1).padStart(2, '0')}</div>
          <h1>{slide.title}</h1>
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
