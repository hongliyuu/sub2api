import { ref, onMounted, onUnmounted, type Ref } from 'vue'

/**
 * WeChat-style swipe/drag to select rows in a DataTable.
 *
 * Usage:
 *   const containerRef = ref<HTMLElement | null>(null)
 *   useSwipeSelect(containerRef, {
 *     isSelected: (id) => selIds.value.includes(id),
 *     select: (id) => { if (!selIds.value.includes(id)) selIds.value.push(id) },
 *     deselect: (id) => { selIds.value = selIds.value.filter(x => x !== id) },
 *   })
 *
 * Wrap <DataTable> with <div ref="containerRef">...</div>
 * DataTable rows must have data-row-id attribute.
 */
export interface SwipeSelectAdapter {
  isSelected: (id: number) => boolean
  select: (id: number) => void
  deselect: (id: number) => void
}

export function useSwipeSelect(
  containerRef: Ref<HTMLElement | null>,
  adapter: SwipeSelectAdapter
) {
  const isDragging = ref(false)

  let dragMode: 'select' | 'deselect' = 'select'
  let startRowIndex = -1
  let lastEndIndex = -1
  // Snapshot of which row IDs were selected when drag started
  let initialSelectedSnapshot = new Map<number, boolean>()
  // Cache of row elements for the current drag operation
  let cachedRows: HTMLElement[] = []

  function getDataRows(): HTMLElement[] {
    const container = containerRef.value
    if (!container) return []
    return Array.from(container.querySelectorAll('tbody tr[data-row-id]'))
  }

  function getRowId(el: HTMLElement): number | null {
    const raw = el.getAttribute('data-row-id')
    if (raw === null) return null
    const id = Number(raw)
    return Number.isFinite(id) ? id : null
  }

  function applyRange(endIndex: number) {
    const rangeMin = Math.min(startRowIndex, endIndex)
    const rangeMax = Math.max(startRowIndex, endIndex)
    const prevMin = lastEndIndex >= 0 ? Math.min(startRowIndex, lastEndIndex) : rangeMin
    const prevMax = lastEndIndex >= 0 ? Math.max(startRowIndex, lastEndIndex) : rangeMax

    // Determine the full affected region (union of old and new range)
    const lo = Math.min(rangeMin, prevMin)
    const hi = Math.max(rangeMax, prevMax)

    for (let i = lo; i <= hi && i < cachedRows.length; i++) {
      const id = getRowId(cachedRows[i])
      if (id === null) continue

      if (i >= rangeMin && i <= rangeMax) {
        // In current range → apply drag mode
        if (dragMode === 'select') {
          adapter.select(id)
        } else {
          adapter.deselect(id)
        }
      } else {
        // Outside current range → restore to initial state
        const wasSelected = initialSelectedSnapshot.get(id) ?? false
        if (wasSelected) {
          adapter.select(id)
        } else {
          adapter.deselect(id)
        }
      }
    }

    lastEndIndex = endIndex
  }

  function onMouseDown(e: MouseEvent) {
    if (e.button !== 0) return

    const target = e.target as HTMLElement
    // Don't interfere with interactive elements
    if (target.closest('button, a, input, select, textarea, [role="button"], [role="menuitem"]')) return

    // Must be inside tbody
    if (!target.closest('tbody')) return

    cachedRows = getDataRows()
    const tr = target.closest('tr[data-row-id]') as HTMLElement | null
    if (!tr) return
    const rowIndex = cachedRows.indexOf(tr)
    if (rowIndex < 0) return

    const rowId = getRowId(tr)
    if (rowId === null) return

    // Snapshot current selection state for all visible rows
    initialSelectedSnapshot = new Map()
    for (const row of cachedRows) {
      const id = getRowId(row)
      if (id !== null) {
        initialSelectedSnapshot.set(id, adapter.isSelected(id))
      }
    }

    isDragging.value = true
    startRowIndex = rowIndex
    lastEndIndex = -1
    dragMode = adapter.isSelected(rowId) ? 'deselect' : 'select'

    applyRange(rowIndex)

    e.preventDefault()
    document.body.style.userSelect = 'none'
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
  }

  function onMouseMove(e: MouseEvent) {
    if (!isDragging.value) return

    const el = document.elementFromPoint(e.clientX, e.clientY) as HTMLElement | null
    if (!el) return

    const tr = el.closest('tr[data-row-id]') as HTMLElement | null
    if (!tr) return
    const rowIndex = cachedRows.indexOf(tr)
    if (rowIndex < 0) return

    applyRange(rowIndex)

    // Auto-scroll when near container edges
    autoScroll(e)
  }

  function onMouseUp() {
    isDragging.value = false
    startRowIndex = -1
    lastEndIndex = -1
    cachedRows = []
    initialSelectedSnapshot.clear()
    stopAutoScroll()
    document.body.style.userSelect = ''

    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  // --- Auto-scroll logic ---
  let scrollRAF = 0
  const SCROLL_ZONE = 40 // px from edge
  const SCROLL_SPEED = 8 // px per frame

  function autoScroll(e: MouseEvent) {
    cancelAnimationFrame(scrollRAF)
    const container = containerRef.value
    if (!container) return

    const rect = container.getBoundingClientRect()
    let dy = 0
    if (e.clientY < rect.top + SCROLL_ZONE) {
      dy = -SCROLL_SPEED
    } else if (e.clientY > rect.bottom - SCROLL_ZONE) {
      dy = SCROLL_SPEED
    }

    if (dy !== 0) {
      const step = () => {
        container.scrollTop += dy
        scrollRAF = requestAnimationFrame(step)
      }
      scrollRAF = requestAnimationFrame(step)
    }
  }

  function stopAutoScroll() {
    cancelAnimationFrame(scrollRAF)
  }

  onMounted(() => {
    containerRef.value?.addEventListener('mousedown', onMouseDown)
  })

  onUnmounted(() => {
    containerRef.value?.removeEventListener('mousedown', onMouseDown)
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
    stopAutoScroll()
  })

  return { isDragging }
}
