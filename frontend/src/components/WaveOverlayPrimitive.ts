import type {
  AutoscaleInfo,
  IChartApiBase,
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesApi,
  ISeriesPrimitive,
  SeriesAttachedParameter,
  Time,
  UTCTimestamp,
} from 'lightweight-charts'
import type { Invalidation, Pivot, Scenario, TargetZone, WaveNode } from '../types/api'

interface OverlayModel {
  scenario: Scenario | null
  comparison: Scenario | null
  futureBars: number[]
  visibleDegrees: string[]
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

function pivotLabels(node: WaveNode): string[] {
  if (node.pattern.includes('TRIANGLE')) return ['A', 'B', 'C', 'D', 'E']
  if (node.pattern.includes('ZIGZAG') || node.pattern.includes('FLAT')) return ['A', 'B', 'C']
  if (node.pattern.includes('THREE')) return ['W', 'X', 'Y', 'X', 'Z']
  return ['1', '2', '3', '4', '5']
}

function degreeLabel(label: string, degree: string): string {
  const roman = label.replace('1', 'i').replace('2', 'ii').replace('3', 'iii').replace('4', 'iv').replace('5', 'v')
  switch (degree) {
    case 'GRAND_SUPERCYCLE': return `[[${label}]]`
    case 'SUPERCYCLE': return `[${label}]`
    case 'CYCLE': return label
    case 'PRIMARY': return `[${label}]`
    case 'INTERMEDIATE': return `(${label})`
    case 'MINOR': return label
    case 'MINUTE': return roman
    case 'MINUETTE': return `(${roman})`
    case 'SUBMINUETTE': return roman
    default: return label
  }
}

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
    const values: number[] = []
    const collectScenario = (scenario: Scenario | null) => {
      if (!scenario) return
      collectNodePrices(scenario.root, values)
      for (const invalidation of scenario.invalidations) {
        if (invalidation.price !== undefined) values.push(invalidation.price)
      }
      for (const target of scenario.target_ladder) {
        if (target.status !== 'INVALIDATED') values.push(target.min_price, target.max_price)
      }
    }
    collectScenario(this.model.scenario)
    collectScenario(this.model.comparison)
    if (values.length === 0) return null
    return {
      priceRange: { minValue: Math.min(...values), maxValue: Math.max(...values) },
      margins: { above: 24, below: 24 },
    }
  }

  draw(context: CanvasRenderingContext2D, width: number, height: number): void {
    if (!this.chart || !this.series) return
    context.save()
    this.drawScenario(context, this.model.comparison, COMPARISON, 0.32, width, height, true)
    this.drawScenario(context, this.model.scenario, PURPLE, 1, width, height, false)
    context.restore()
  }

  private drawScenario(
    context: CanvasRenderingContext2D,
    scenario: Scenario | null,
    color: string,
    opacity: number,
    width: number,
    height: number,
    comparison: boolean,
  ): void {
    if (!scenario) return
    const labels: LabelPoint[] = []
    this.drawNode(context, scenario.root, color, opacity, 0, labels)
    if (!comparison) {
      this.drawTargets(context, scenario.target_ladder, scenario.root.orthodox_end.time, opacity, width)
      this.drawInvalidations(context, scenario.invalidations, scenario.root.orthodox_end.time, opacity, width)
    }
    this.drawLabels(context, labels, height)
  }

  private drawNode(
    context: CanvasRenderingContext2D,
    node: WaveNode,
    color: string,
    opacity: number,
    depth: number,
    labels: LabelPoint[],
  ): void {
    const visible = this.model.visibleDegrees.length === 0 ||
      this.model.visibleDegrees.includes(node.degree)
    const coordinates = node.pivots
      .map((pivot) => ({ pivot, point: this.coordinate(pivot) }))
      .filter((entry): entry is { pivot: Pivot; point: DrawPoint } => entry.point !== null)
    if (visible && coordinates.length >= 2) {
      context.beginPath()
      context.strokeStyle = colorWithOpacity(color, opacity * Math.max(0.3, 1 - depth * 0.18))
      context.lineWidth = Math.max(1, 3 - depth * 0.55)
      context.setLineDash(node.status === 'DEVELOPING' ? [7, 4] : [])
      context.moveTo(coordinates[0].point.x, coordinates[0].point.y)
      for (const entry of coordinates.slice(1)) {
        context.lineTo(entry.point.x, entry.point.y)
      }
      context.stroke()
      context.setLineDash([])

      const names = pivotLabels(node)
      coordinates.slice(1).forEach((entry, index) => {
        if (index >= names.length) return
        labels.push({
          ...entry.point,
          text: degreeLabel(names[index], node.degree),
          color,
          opacity: opacity * Math.max(0.4, 1 - depth * 0.18),
        })
      })
    }
    for (const child of node.children ?? []) {
      this.drawNode(context, child, color, opacity, depth + 1, labels)
    }
  }

  private drawTargets(
    context: CanvasRenderingContext2D,
    targets: TargetZone[],
    anchorTime: number,
    opacity: number,
    width: number,
  ): void {
    const anchorX = this.timeCoordinate(anchorTime) ?? Math.max(0, width - 260)
    for (const target of targets) {
      if (target.status === 'INVALIDATED') continue
      if (target.geometry === 'CHANNEL_POLYGON' && (target.points?.length ?? 0) >= 3) {
        const polygon = target.points
          ?.map((point) => {
            const time = point.bar_offset <= 0
              ? anchorTime
              : this.model.futureBars[point.bar_offset - 1]
            const x = time === undefined ? null : this.timeCoordinate(time)
            const y = this.series?.priceToCoordinate(point.price)
            return x === null || y === null || y === undefined ? null : { x, y: Number(y) }
          })
          .filter((point): point is DrawPoint => point !== null) ?? []
        if (polygon.length >= 3) {
          context.beginPath()
          context.moveTo(polygon[0].x, polygon[0].y)
          for (const point of polygon.slice(1)) context.lineTo(point.x, point.y)
          context.closePath()
          context.fillStyle = `rgba(158, 72, 255, ${0.07 * opacity})`
          context.strokeStyle = `rgba(190, 118, 255, ${0.52 * opacity})`
          context.setLineDash([7, 5])
          context.fill()
          context.stroke()
          context.setLineDash([])
          context.fillStyle = `rgba(227, 204, 255, ${0.82 * opacity})`
          context.font = '600 10px Inter, system-ui, sans-serif'
          context.fillText(target.wave_label, polygon[0].x + 6, polygon[0].y - 5)
        }
        continue
      }
      const y1 = this.series?.priceToCoordinate(target.min_price)
      const y2 = this.series?.priceToCoordinate(target.max_price)
      if (y1 === null || y1 === undefined || y2 === null || y2 === undefined) continue
      const endTime = target.time_window?.end_time ??
        this.model.futureBars[Math.min(49, this.model.futureBars.length - 1)]
      const endX = endTime ? (this.timeCoordinate(endTime) ?? width) : width
      const lineOnly = target.confluence === 'SINGLE_LEVEL' || Math.abs(target.max_price - target.min_price) < 1e-9
      context.beginPath()
      if (lineOnly) {
        context.strokeStyle = `rgba(190, 118, 255, ${0.78 * opacity})`
        context.lineWidth = 1
        context.setLineDash([4, 4])
        context.moveTo(anchorX, Number(y1))
        context.lineTo(endX, Number(y1))
        context.stroke()
      } else {
        const top = Math.min(Number(y1), Number(y2))
        const boxHeight = Math.max(2, Math.abs(Number(y1) - Number(y2)))
        context.fillStyle = target.confluence === 'HIGH'
          ? `rgba(158, 72, 255, ${0.22 * opacity})`
          : `rgba(158, 72, 255, ${0.13 * opacity})`
        context.strokeStyle = `rgba(190, 118, 255, ${0.82 * opacity})`
        context.lineWidth = target.confluence === 'HIGH' ? 2 : 1
        context.setLineDash(target.status === 'CONDITIONAL' ? [6, 4] : [])
        context.rect(anchorX, top, Math.max(2, endX - anchorX), boxHeight)
        context.fill()
        context.stroke()
      }
      context.setLineDash([])
      context.fillStyle = `rgba(227, 204, 255, ${0.9 * opacity})`
      context.font = '600 11px Inter, system-ui, sans-serif'
      context.fillText(`${target.wave_label} · ${target.confluence}`, anchorX + 6, Math.min(Number(y1), Number(y2)) - 5)
    }
  }

  private drawInvalidations(
    context: CanvasRenderingContext2D,
    invalidations: Invalidation[],
    anchorTime: number,
    opacity: number,
    width: number,
  ): void {
    const startX = this.timeCoordinate(anchorTime) ?? 0
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
      context.font = '500 10px Inter, system-ui, sans-serif'
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

  private coordinate(pivot: Pivot): DrawPoint | null {
    const x = this.timeCoordinate(pivot.time)
    const y = this.series?.priceToCoordinate(pivot.price)
    if (x === null || y === null || y === undefined) return null
    return { x, y: Number(y) }
  }

  private timeCoordinate(time: number): number | null {
    const coordinate = this.chart?.timeScale().timeToCoordinate(time as UTCTimestamp)
    return coordinate === null || coordinate === undefined ? null : Number(coordinate)
  }
}

function collectNodePrices(node: WaveNode, values: number[]): void {
  values.push(...node.pivots.map((pivot) => pivot.price))
  for (const child of node.children ?? []) collectNodePrices(child, values)
}

function colorWithOpacity(hex: string, opacity: number): string {
  const normalized = hex.replace('#', '')
  const value = Number.parseInt(normalized, 16)
  const red = (value >> 16) & 255
  const green = (value >> 8) & 255
  const blue = value & 255
  return `rgba(${red}, ${green}, ${blue}, ${Math.max(0, Math.min(1, opacity))})`
}
