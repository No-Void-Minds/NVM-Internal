// Test unit pengenal jenis tautan output Aktivitas (revisi 21 lanjutan).
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import { driveThumbnail, outputKind } from '../src/utils/outputLink.ts'

test('lima pola Google dikenali beserta labelnya', () => {
  const cases: [string, string, string][] = [
    ['https://drive.google.com/file/d/1AbC-d_E/view?usp=sharing', 'drive-file', 'File Drive'],
    ['https://drive.google.com/drive/folders/0Bx_yZ-9?usp=drive_link', 'drive-folder', 'Folder Drive'],
    ['https://drive.google.com/drive/u/1/folders/0Bx', 'drive-folder', 'Folder Drive'],
    ['https://docs.google.com/document/d/abc/edit#heading=h.1', 'docs', 'Google Docs'],
    ['https://docs.google.com/spreadsheets/d/abc/edit?gid=0#gid=0', 'sheets', 'Spreadsheet'],
    ['https://docs.google.com/presentation/d/abc/edit', 'slides', 'Slide'],
  ]
  for (const [url, kind, label] of cases) {
    const k = outputKind(url)
    assert.equal(k?.kind, kind, url)
    assert.equal(k?.label, label, url)
  }
})

test('id file Drive diambil tanpa parameter, dan dipakai untuk thumbnail', () => {
  const k = outputKind('https://drive.google.com/file/d/1AbC-d_E/view?usp=sharing')
  assert.equal(k?.driveFileId, '1AbC-d_E')
  assert.equal(driveThumbnail('1AbC-d_E'), 'https://drive.google.com/thumbnail?id=1AbC-d_E&sz=w480')
  assert.equal(outputKind('https://docs.google.com/document/d/abc/edit')?.driveFileId, null)
})

test('http tetap dikenali, host lain memakai nama host', () => {
  assert.equal(outputKind('http://drive.google.com/file/d/xyz/view')?.kind, 'drive-file')
  const other = outputKind('https://www.canva.com/design/abc/edit')
  assert.deepEqual(other, { kind: 'host', label: 'www.canva.com', driveFileId: null })
  // Path Drive lain dan host tiruan tidak dianggap Google.
  assert.equal(outputKind('https://drive.google.com/open?id=abc')?.kind, 'host')
  assert.equal(outputKind('https://docs.google.com.evil.test/document/d/x')?.kind, 'host')
})

test('bukan tautan tidak punya jenis', () => {
  assert.equal(outputKind('Naskah final di folder tim'), null)
  assert.equal(outputKind('javascript:alert(1)'), null)
  assert.equal(outputKind(''), null)
})
