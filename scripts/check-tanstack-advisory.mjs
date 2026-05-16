import fs from 'node:fs'

const blocked = new Set([
  '@tanstack/react-router@1.169.5',
  '@tanstack/react-router@1.169.8',
  '@tanstack/router-core@1.169.5',
  '@tanstack/router-core@1.169.8',
  '@tanstack/router-plugin@1.169.5',
  '@tanstack/router-plugin@1.169.8',
  '@tanstack/router-vite-plugin@1.169.5',
  '@tanstack/router-vite-plugin@1.169.8',
])

const files = ['package.json', 'pnpm-lock.yaml'].filter((file) =>
  fs.existsSync(file),
)
const haystack = files.map((file) => fs.readFileSync(file, 'utf8')).join('\n')

const found = [...blocked].filter((pkg) => {
  const [name, version] = pkg.split(/@(?=\\d)/)
  return haystack.includes(name) && haystack.includes(version)
})

if (found.length > 0) {
  console.error('Blocked TanStack package versions from GHSA-g7cv-rxg3-hmpx:')
  for (const item of found) console.error(`- ${item}`)
  process.exit(1)
}
