import type {
  AutoscaleInfo,
  IChartApiBase,
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesApi,
  ISeriesPrimitive,
  PrimitiveHoveredItem,
  SeriesAttachedParameter,
  Time,
  UTCTimestamp,
} from 'lightweight-charts'
import type {
  CanonicalWaveEvent,
  Invalidation,
  MasterScenario,
  MasterWaveGraph,
  MasterWaveNode,
  TargetZone,
  TimeframeView,
} from '../types/api'

export interface OverlayModel {
  graph: MasterWaveGraph | null
  view: TimeframeView | null
  scenario: MasterScenario | null
  comparison: MasterScenario | null
  selectedNodeID: string
}

interface DrawPoint {
  x: number
  y: number
}

interface LabelPoint extends DrawPoint {
  text: string
  color: string
  opacity: number
}

const PURPLE = '#b874ff'
const COMPARISON = '#7183ad'

class OverlayRenderer implements IPrimitivePaneRenderer {
  private readonly source: WaveOverlayPrimitive

  constructor(source: WaveOverlayPrimitive) {
    this.source = source
  }

  draw(target: Parameters<IPrimitivePaneRenderer['draw']>[0]): void {
    target.useMediaCoordinateSpace((scope) => {
      this.source.draw(scope.context, scope.mediaSize.width, scope.mediaSize.height)
    })
  }
}

class OverlayPaneView implements IPrimitivePaneView {
  private readonly rendererValue: OverlayRenderer

  constructor(source: WaveOverlayPrimitive) {
    this.rendererValue = new OverlayRenderer(source)
  }

  zOrder(): 'top' {
    return 'top'
  }

  renderer(): IPrimitivePaneRenderer {
    return this.rendererValue
  }
}

export class WaveOverlayPrimitive implements ISeriesPrimitive<Time> {
  private chart: IChartApiBase<Time> | null = null
  private series: ISeriesApi<'Candlestick', Time> | null = null
  private requestUpdate: (() => void) | null = null
  private model: OverlayModel
  private readonly views: readonly IPrimitivePaneView[]

  constructor(model: OverlayModel) {
    this.model = model
    this.views = [new OverlayPaneView(this)]
  }

  attached(parameters: SeriesAttachedParameter<Time, 'Candlestick'>): void {
    this.chart = parameters.chart
    this.series = parameters.series
    this.requestUpdate = parameters.requestUpdate
  }

  detached(): void {
    this.chart = null
    this.series = null
    this.requestUpdate = null
  }

  paneViews(): readonly IPrimitivePaneView[] {
    return this.views
  }

  update(model: OverlayModel): void {
    this.model = model
    this.requestUpdate?.()
  }

  updateAllViews(): void {
    this.requestUpdate?.()
  }

  autoscaleInfo(): AutoscaleInfo | null {
    const graph = this.model.graph
    if (!graph) return null
    const values = graph.events.map((event) => event.orthodox_price)
    for (const scenario of [this.model.scenario, this.model.comparison]) {
      if (!scenario) continue
      for (const invalidation of scenario.invalidations) {
        if (invalidation.price !== undefined) values.push(invalidation.price)
      }
      for (const target of scenario.target_ladder) {
        if (target.status !== 'INVALIDATED') values.push(target.min_price, target.max_price)
      }
    }
    if (values.length === 0) return null
    return {
      priceRange: { minValue: Math.min(...values), maxValue: Math.max(...values) },
      margins: { above: 24, below: 24 },
    }
  }

  hitTest(x: number, y: number): PrimitiveHoveredItem | null {
    const graph = this.model.graph
    const view = this.model.view
    const scenario = this.model.scenario
    if (!graph || !view || !scenario || !this.series) return null
    const nodeByID = new Map(graph.nodes.map((node) => [node.id, node]))
    const eventByID = new Map(graph.events.map((event) => [event.id, event]))
    const allowed = scenarioNodeIDs(scenario, nodeByID)
    const ids = [...new Set([...view.visible_node_ids, ...view.ancestor_node_ids])]
    let best: { id: string; distance: number } | null = null
    for (const id of ids) {
      if (!allowed.has(id)) continue
      const node = nodeByID.get(id)
      if (!node) continue
      const points = node.pivot_event_ids
        .map((eventID) => eventByID.get(eventID))
        .map((event) => event ? this.coordinate(event) : null)
        .filter((point): point is DrawPoint => point !== null)
      for (let index = 1; index < points.length; index++) {
        const distance = distanceToSegment({ x, y }, points[index - 1], points[index])
        if (distance <= 8 && (!best || distance < best.distance)) best = { id, distance }
      }
    }
    return best ? {
      externalId: best.id,
      distance: best.distance,
      hitTestPriority: 1,
      cursorStyle: 'pointer',
      zOrder: 'top',
      itemType: 'primitive',
    } : null
  }

  draw(context: CanvasRenderingContext2D, width: number, height: number): void {
    if (!this.chart || !this.series || !this.model.graph || !this.model.view) return
    context.save()
    this.drawScenario(context, this.model.comparison, COMPARISON, 0.3, width, height, true)
    this.drawScenario(context, this.model.scenario, PURPLE, 1, width, height, false)
    context.restore()
  }

  private drawScenario(
    context: CanvasRenderingContext2D,
    scenario: MasterScenario | null,
    color: string,
    opacity: number,
    width: number,
    height: number,
    comparison: boolean,
  ): void {
    const graph = this.model.graph
    const view = this.model.view
    if (!scenario || !graph || !view) return
    const nodeByID = new Map(graph.nodes.map((node) => [node.id, node]))
    const eventByID = new Map(graph.events.map((event) => [event.id, event]))
    const allowed = scenarioNodeIDs(scenario, nodeByID)
    const ancestorSet = new Set(view.ancestor_node_ids)
    const renderIDs = [...new Set([...view.visible_node_ids, ...view.ancestor_node_ids])]
      .filter((id) => allowed.has(id))
      .sort((left, right) => {
        const leftNode = nodeByID.get(left)
        const rightNode = nodeByID.get(right)
        return (rightNode?.pivot_event_ids.length ?? 0) - (leftNode?.pivot_event_ids.length ?? 0)
      })
    const labels: LabelPoint[] = []
    for (const id of renderIDs) {
      const node = nodeByID.get(id)
      if (!node) continue
      this.drawNode(
        context,
        node,
        eventByID,
        color,
        opacity,
        ancestorSet.has(id),
        id === this.model.selectedNodeID,
        labels,
      )
    }
    if (!comparison) {
      const anchor = activeAnchor(scenario, nodeByID, eventByID)
      this.drawTargets(context, scenario.target_ladder, anchor, opacity, width)
      this.drawInvalidations(context, scenario.invalidations, anchor, opacity, width)
    }
    this.drawLabels(context, labels, height)
  }

  private drawNode(
    context: CanvasRenderingContext2D,
    node: MasterWaveNode,
    events: Map<string, CanonicalWaveEvent>,
    color: string,
    opacity: number,
    ancestor: boolean,
    selected: boolean,
    labels: LabelPoint[],
  ): void {
    const coordinates = node.pivot_event_ids
      .map((id) => {
        const event = events.get(id)
        return event ? { event, point: this.coordinate(event) } : null
      })
      .filter((entry): entry is { event: CanonicalWaveEvent; point: DrawPoint } =>
        entry !== null && entry.point !== null)
    if (coordinates.length < 2) return
    const lineOpacity = opacity * (ancestor ? 0.48 : 0.9)
    context.beginPath()
    context.strokeStyle = colorWithOpacity(color, selected ? opacity : lineOpacity)
    context.lineWidth = selected ? 3.5 : ancestor ? 2.5 : 1.35
    context.setLineDash(node.status === 'DEVELOPING' ? [7, 4] : [])
    context.moveTo(coordinates[0].point.x, coordinates[0].point.y)
    for (const entry of coordinates.slice(1)) context.lineTo(entry.point.x, entry.point.y)
    context.stroke()
    context.setLineDash([])

    const names = pivotLabels(node)
    coordinates.slice(1).forEach((entry, index) => {
      if (index >= names.length || (ancestor && !selected && index < coordinates.length - 2)) return
      labels.push({
        ...entry.point,
        text: degreeLabel(names[index], node.degree),
        color,
        opacity: selected ? opacity : lineOpacity,
      })
    })
  }

  private drawTargets(
    context: CanvasRenderingContext2D,
    targets: TargetZone[],
    anchor: CanonicalWaveEvent | null,
    opacity: number,
    width: number,
  ): void {
    const view = this.model.view
    if (!view) return
    const anchorX = anchor ? (this.timeCoordinate(this.chartTime(anchor)) ?? Math.max(0, width - 260)) : Math.max(0, width - 260)
    for (const target of targets) {
      if (target.status === 'INVALIDATED') continue
      const y1 = this.series?.priceToCoordinate(target.min_price)
      const y2 = this.series?.priceToCoordinate(target.max_price)
      if (y1 === null || y1 === undefined || y2 === null || y2 === undefined) continue
      const endTime = target.time_window?.end_time ??
        view.future_logical_bars[Math.min(49, view.future_logical_bars.length - 1)]
      const endX = endTime ? (this.timeCoordinate(endTime) ?? width) : width
      const lineOnly = target.confluence === 'SINGLE_LEVEL' ||
        Math.abs(target.max_price - target.min_price) < 1e-9
      context.beginPath()
      context.strokeStyle = `rgba(190, 118, 255, ${0.82 * opacity})`
      context.setLineDash(target.status === 'CONDITIONAL' ? [6, 4] : [])
      if (lineOnly) {
        context.moveTo(anchorX, Number(y1))
        context.lineTo(endX, Number(y1))
        context.stroke()
      } else {
        const top = Math.min(Number(y1), Number(y2))
        context.fillStyle = target.confluence === 'HIGH'
          ? `rgba(158, 72, 255, ${0.22 * opacity})`
          : `rgba(158, 72, 255, ${0.13 * opacity})`
        context.lineWidth = target.confluence === 'HIGH' ? 2 : 1
        context.rect(anchorX, top, Math.max(2, endX - anchorX), Math.max(2, Math.abs(Number(y1) - Number(y2))))
        context.fill()
        context.stroke()
      }
      context.setLineDash([])
      context.fillStyle = `rgba(227, 204, 255, ${0.9 * opacity})`
      context.font = '600 10px Inter, system-ui, sans-serif'
      context.fillText(`${target.wave_label} · ${target.confluence}`, anchorX + 6, Math.min(Number(y1), Number(y2)) - 5)
    }
  }

  private drawInvalidations(
    context: CanvasRenderingContext2D,
    invalidations: Invalidation[],
    anchor: CanonicalWaveEvent | null,
    opacity: number,
    width: number,
  ): void {
    const startX = anchor ? (this.timeCoordinate(this.chartTime(anchor)) ?? 0) : 0
    for (const invalidation of invalidations) {
      if (invalidation.price === undefined) continue
      const y = this.series?.priceToCoordinate(invalidation.price)
      if (y === null || y === undefined) continue
      context.beginPath()
      context.strokeStyle = `rgba(255, 102, 125, ${0.72 * opacity})`
      context.lineWidth = 1
      context.setLineDash([3, 5])
      context.moveTo(startX, Number(y))
      context.lineTo(width, Number(y))
      context.stroke()
      context.setLineDash([])
      context.fillStyle = `rgba(255, 151, 166, ${0.9 * opacity})`
      context.font = '500 9px Inter, system-ui, sans-serif'
      context.fillText('INVALIDATION', startX + 6, Number(y) - 4)
    }
  }

  private drawLabels(context: CanvasRenderingContext2D, labels: LabelPoint[], height: number): void {
    const occupied: Array<{ x: number; y: number }> = []
    for (const label of labels) {
      let y = label.y - 10
      while (occupied.some((point) => Math.abs(point.x - label.x) < 34 && Math.abs(point.y - y) < 18)) {
        y -= 18
      }
      y = Math.max(14, Math.min(height - 8, y))
      occupied.push({ x: label.x, y })
      context.fillStyle = colorWithOpacity('#080a12', 0.82 * label.opacity)
      context.strokeStyle = colorWithOpacity(label.color, 0.65 * label.opacity)
      context.lineWidth = 1
      context.font = '700 11px "JetBrains Mono", monospace'
      const textWidth = context.measureText(label.text).width
      context.beginPath()
      context.roundRect(label.x - textWidth / 2 - 4, y - 11, textWidth + 8, 16, 4)
      context.fill()
      context.stroke()
      context.fillStyle = colorWithOpacity('#f4eaff', label.opacity)
      context.textAlign = 'center'
      context.fillText(label.text, label.x, y)
    }
    context.textAlign = 'start'
  }

  private coordinate(event: CanonicalWaveEvent): DrawPoint | null {
    const x = this.timeCoordinate(this.chartTime(event))
    const y = this.series?.priceToCoordinate(event.orthodox_price)
    if (x === null || y === null || y === undefined) return null
    return { x, y: Number(y) }
  }

  private chartTime(event: CanonicalWaveEvent): number {
    const candles = this.model.view?.candles ?? []
    let low = 0
    let high = candles.length
    while (low < high) {
      const middle = Math.floor((low + high) / 2)
      if (candles[middle].source_to < event.orthodox_time) low = middle + 1
      else high = middle
    }
    const candle = candles[low]
    if (candle && candle.source_from <= event.orthodox_time && candle.source_to >= event.orthodox_time) {
      return candle.time
    }
    return event.orthodox_time
  }

  private timeCoordinate(time: number): number | null {
    const coordinate = this.chart?.timeScale().timeToCoordinate(time as UTCTimestamp)
    return coordinate === null || coordinate === undefined ? null : Number(coordinate)
  }
}

function scenarioNodeIDs(
  scenario: MasterScenario,
  nodes: Map<string, MasterWaveNode>,
): Set<string> {
  const result = new Set<string>()
  const include = (id: string): void => {
    if (result.has(id)) return
    const node = nodes.get(id)
    if (!node) return
    result.add(id)
    node.child_ids.forEach(include)
  }
  scenario.observation_root.context_sequence.forEach(include)
  scenario.active_path.forEach(include)
  return result
}

function activeAnchor(
  scenario: MasterScenario,
  nodes: Map<string, MasterWaveNode>,
  events: Map<string, CanonicalWaveEvent>,
): CanonicalWaveEvent | null {
  const activeID = scenario.active_path.at(-1)
  const node = activeID ? nodes.get(activeID) : undefined
  return node ? events.get(node.end_event_id) ?? null : null
}

function pivotLabels(node: MasterWaveNode): string[] {
  if (node.pattern.includes('TRIANGLE')) return ['A', 'B', 'C', 'D', 'E']
  if (node.pattern.includes('ZIGZAG') || node.pattern.includes('FLAT')) return ['A', 'B', 'C']
  if (node.pattern.includes('THREE')) return ['W', 'X', 'Y', 'X', 'Z']
  return ['1', '2', '3', '4', '5']
}

function degreeLabel(label: string, degree: string): string {
  const roman = label.replace('1', 'i').replace('2', 'ii').replace('3', 'iii').replace('4', 'iv').replace('5', 'v')
  switch (degree) {
    case 'GRAND_SUPERCYCLE': return `[[${label}]]`
    case 'SUPERCYCLE':
    case 'PRIMARY': return `[${label}]`
    case 'INTERMEDIATE': return `(${label})`
    case 'MINUTE': return roman
    case 'MINUETTE': return `(${roman})`
    case 'SUBMINUETTE': return roman
    default: return label
  }
}

function colorWithOpacity(hex: string, opacity: number): string {
  const value = Number.parseInt(hex.replace('#', ''), 16)
  return `rgba(${(value >> 16) & 255}, ${(value >> 8) & 255}, ${value & 255}, ${Math.max(0, Math.min(1, opacity))})`
}

function distanceToSegment(point: DrawPoint, start: DrawPoint, end: DrawPoint): number {
  const dx = end.x - start.x
  const dy = end.y - start.y
  if (dx === 0 && dy === 0) return Math.hypot(point.x - start.x, point.y - start.y)
  const ratio = Math.max(0, Math.min(1, ((point.x - start.x) * dx + (point.y - start.y) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(point.x - (start.x + ratio * dx), point.y - (start.y + ratio * dy))
}
