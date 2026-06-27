# MASTER SYSTEM CONTEXT & COMPREHENSIVE FRACTAL ROADMAP: PROJECT WAVESIGHT

## 1. Project Overzicht & Visie
WaveSight is een ultra-high-performance, geautomatiseerde analyse-engine voor Elliott Wave en Fibonacci-fractals. Het doel van het systeem is het elimineren van menselijke subjectiviteit bij het tellen van golven door realtime marktdata (koersdata) te ontleden in wiskundig gevalideerde, recursieve boomstructuren conform de wetten van de *R.N. Elliott Wave Principle*.

### Kernfilosofie:
- **Absolute Recursie:** De markt is niet plat. Het systeem verwerkt koersdata als een hiërarchische boomstructuur (Fractal Tree) waarin grotere trends (Parent Waves) verplicht worden gevalideerd door de interne patronen van een lagere graad (Child Sub-waves).
- **Wiskundige Hardheid:** Strikt onderscheid tussen *Kardinale Wetten* (nul-tolerantie, directe uitsluiting bij overtreding) en *Fibonacci Richtlijnen* (probabilistische scoring via vloeibare linear decay zones).
- **Architectonische Zuiverheid:** Extreme geheugenoptimalisatie (O(N) single-pass execution), nul onnodige heap-allocaties in de Go-pijplijn en asynchrone parallelle dataverwerking.

---

## 2. Tech Stack & Systeemarchitectuur
- **Backend Core:** Go (Golang 1.22+) – Maakt exclusief gebruik van native method-based routing via `http.NewServeMux()`. Systeemonderdelen communiceren via Dependency Inversion (DIP).
- **High-Speed Serialisatie:** `mailru/easyjson` code-generatie om reflectie-vrije, bliksemsnelle JSON-Marshalling van diep geneste boomstructuren te garanderen.
- **Database Caching:** SQLite via `github.com/glebarez/go-sqlite` (Pure Go, CGO-free) voor performante opslag en indexering van historische kaarsdata.
- **Frontend Canvas:** Vue 3 (Composition API, `<script setup lang="ts">`, Vite) – Maakt gebruik van `shallowRef` en `raw arrays` om CPU-lag bij duizenden actieve datapunten te elimineren.
- **Grafische Grafiek-rendering:** TradingView Lightweight Charts (`lightweight-charts`) met op maat gemaakte Canvas Primitives voor schuine geometrische projecties en doelvakken.

---

## 3. Uitputtende Catalogus van Elliott Wave Structuren
De engine valideert de markt tegen de complete theoretische catalogus. Elk patroon is gedefinieerd met harde wiskundige randvoorwaarden.

### 3.1 Motive Waves (Trendgolven - Cijfernotatie 1-2-3-4-5)
Motive waves bewegen áltijd in de richting van de grotere trend van één graad groter.

#### A. Motive Impulse (Klassieke Impulsgolf)
- **Interne Fractal Structuur:** 5-3-5-3-5 (Golven 1, 3 en 5 zijn zelf Motive; Golven 2 en 4 zijn Corrective).
- **Kardinale Wet 1:** Wave 2 retraceert nooit meer dan 100% van Wave 1.
- **Kardinale Wet 2:** Wave 3 is wiskundig nooit de kortste van de drie actieve golven (1, 3 en 5).
- **Kardinale Wet 3:** Wave 4 mag het prijsgebied van Wave 1 nooit betreden of overlappen (prijslimiet nul-tolerantie).
- **Extensie Richtlijn:** Minimaal één van de actieve golven (1, 3 of 5) moet een duidelijke extensie vertonen ($\ge 1.618$ van de basisschaal). Als Wave 3 extended is, zijn Wave 1 en 5 vaak nagenoeg gelijk in prijslengte of verhouden zich tot $0.618$.
- **Alternation Richtlijn:** Wave 2 en Wave 4 moeten van elkaar afwijken in diepte en complexiteit. (Scherpe vs. Zijwaartse/Complexe correctie).
- **Geometrische Channeling:** Een rechte trendlijn over de eindpunten van Wave 2 en Wave 4, geprojecteerd als een parallelle lijn vanaf de top van Wave 3, definieert de exacte schuine landingszone voor Wave 5.

#### B. Leading Diagonal (Aanvangende Diagonaal)
- **Locatie:** Komt uitsluitend voor als Wave 1 van een impuls of Wave A van een zigzag.
- **Interne Fractal Structuur:** Kan zowel 5-3-5-3-5 als 3-3-3-3-3 zijn.
- **Kardinale Regels:** Wave 2 retraceert niet meer dan 100% van Wave 1. Wave 3 is niet de kortste.
- **Geometrische Afwijking:** Wave 4 overlap met Wave 1 is hier verplicht.
- **Converging Sub-type:** De golven vernauwen zich wiskundig: $\text{Lengte W1} > \text{Lengte W3} > \text{Lengte W5}$ en $\text{Lengte W2} > \text{Lengte W4}$.
- **Expanding Sub-type:** De golven verruimen zich wiskundig: $\text{Lengte W1} < \text{Lengte W3} < \text{Lengte W5}$.

#### C. Ending Diagonal (Eindigende Diagonaal / Wedge)
- **Locatie:** Komt uitsluitend voor als Wave 5 van een impuls of Wave C van een zigzag/flat. Kondigt een felle trendomslag aan.
- **Interne Fractal Structuur:** Strikt 3-3-3-3-3 (Elke sub-poot is verplicht een corrigerende ABC-vorm).
- **Geometrische Afwijking:** Wave 4 overlap met Wave 1 is verplicht. Vertoont een wig-vorm (Converging of Expanding).

#### D. Truncated Fifth (Afgeknotte Vijfde Golf)
- **Voorwaarde:** Wave 5 slaagt er wiskundig niet in om voorbij het uiterste prijsextreem van Wave 3 te breken.
- **Validatie-eis:** Mag alleen als valide worden beschouwd door de scorer als Wave 3 een extreme overextensie vertoont ($\ge 2.62$ van Wave 1).

---

### 3.2 Corrective Waves (Correctiegolven - Letternotatie A-B-C / W-X-Y-X-Z)
Corrective waves bewegen áltijd tegen de richting van de grotere trend van één graad groter in.

#### A. Zigzag Correcties (Scherpe, Diepe Correcties)
- **Single Zigzag (ABC):**
  - **Interne Fractal Structuur:** Strikt 5-3-5. (Wave A is een impuls, B is een 3-wave correctie, C is een impuls).
  - **Wiskundige Ratio's:** Wave B retraceert tussen 38.2% en 61.8% van Wave A. Wave C reikt tot $0.62\times, 1.00\times, \text{of } 1.62\times$ van de lengte van Wave A.
- **Double Zigzag (WXY) & Triple Zigzag (WXYXZ):**
  - **Doel:** Wordt gevormd als een Single Zigzag onvoldoende prijs-correctie heeft opgeleverd.
  - **Kardinale Opeenvolging:** Wave X en Wave X2 fungeren als corrigerende verbindingslussen (mogen zelf een kleine zigzag zijn) en mogen niet meer dan 90% van de voorgaande golf heroveren.

#### B. Flat Correcties (Zijwaartse Correcties)
- **Regular Flat:**
  - **Interne Fractal Structuur:** Strikt 3-3-5. (Wave A en B zijn 3-wave correcties, Wave C is een 5-wave impuls/diagonaal).
  - **Wiskundige Ratio's:** Wave B eindigt nagenoeg aan het startpunt van Wave A (90% tot 105% retracement). Wave C eindigt net iets voorbij het eindpunt van Wave A.
- **Expanded Flat (Irregular Flat):**
  - **Wiskundige Ratio's:** Wave B breekt vurig voorbij het startpunt van Wave A ($105\% \text{ tot } 138.2\%$). Wave C maakt vervolgens een gigantische beweging en eindigt ver voorbij het eindpunt van Wave A ($1.236\times \text{ tot } 1.62\times$ van Wave A).
- **Running Flat:**
  - **Wiskundige Ratio's:** Wave B breekt voorbij het startpunt van Wave A, maar Wave C is uitgeput en slaagt er niet in om het eindpunt van Wave A te bereiken (faalt halverwege).

#### C. Triangles (Driehoeken - Letternotatie a-b-c-d-e)
- **Algemene Regels:** Bestaan uit 5 overlappende golven met een interne 3-3-3-3-3 structuur. Mogen uitsluitend voorkomen op de positie van Wave 4, Wave B of de finale Wave Y/Z van een combinatie. **Nooit** als Wave 2 of Wave W.
- **Contracting Triangle:** De trendlijnen getrokken over de toppen $A-C$ en bodems $B-D$ convergeren (lopen wiskundig naar elkaar toe). Ratios van opeenvolgende legs liggen rond de $0.618$.
- **Barrier Triangle:** Gelijk aan Contracting, maar de $B-D$ lijn (of $A-C$ lijn) loopt nagenoeg 100% horizontaal, wat een strakke psychologische steun/weerstand vormt.
- **Expanding Triangle:** De trendlijnen divergeren (lopen wiskundig uit elkaar). Elke opeenvolgende leg is groter dan de vorige ($A < B < C < D < E$).
- **Sferische Sub-types:** Symmetrisch, Ascending (stijgend), Descending (dalend).

#### D. Complexe Combinaties (Double Threes & Triple Threes)
- **Structuur:** Horizontale combinaties gelabeld als $W-X-Y$ en $W-X-Y-X-Z$.
- **Wiskundige Restrictie:** Combineren Flats, Zigzags en Driehoeken. Een `TRIANGLE` mag conform Prechter **nooit** op de positie van Wave W verschijnen; een driehoek mag uitsluitend optreden als de absolute terminale afsluiting (Wave Y of Wave Z).

---

## 4. Gerealiseerde Status & Volledig Fractale Architectuur

### [x] Fase 1: Vloeibare Fibonacci & Statistische Zones (`elliott.go`)
- Harde scheiding tussen kardinale wetten (directe uitsluiting bij overtreding van Wave 2 of Wave 3 limieten) en statistische eSignal-verhoudingen.
- Implementatie van `scoreWave4Continual` om een vloeibaar continuum tussen `0.24` en `0.62` te scoren, waarbij de zone `0.30 - 0.50` de maximale score (`1.0`) krijgt.
- Co-existentie check ingebouwd voor Wave 5 targets: berekent gelijktijdig de verhoudingen tot Wave 1 én de netto afstand van $0\rightarrow3$.

### [x] Fase 2: Geometrische Elliott Parallel Channeling (`elliott.go`)
- Dynamische berekening van de richtingscoëfficiënt (helling $m$) van de trendlijn tussen Pivot 2 en Pivot 4.
- Projectie van parallelle schuine kanalen vanaf de toppen van Wave 1 en Wave 3.
- Bounding-box validatie om schuine kanaalvoorwaarden te vertalen naar frontend-vriendelijke rechthoekige coördinaten.

---

## 5. Fase 3 & 4 Master Roadmap: De Fractal Tree Engine (Vervolgstappen)

### [x] Stap 1: Structuur-gestuurde Dynamic Swing Detection Engine (`pkg/api/api.go` & `pkg/swing/` of `pkg/elliott/`)
- **Probleem:** Het huidige project leunt zwaar op een starre percentage-gebaseerde ZigZag-filterheuristiek (zowel in de huidige code als de eerdere plannen voor multi-depth ZigZag). In de *Elliott Wave Principle* van Frost & Prechter bestaat er echter geen percentage-gebaseerde drempelwaarde om golven te identificeren. Golven zijn schaalvrij (fractal) en worden gedefinieerd door hun interne sub-golfstructuren en onderlinge Fibonacci-verhoudingen. Vaste percentages veroorzaken data-hongersnood voor kleine golven en introduceren ruispivots bij grote golven, wat de recursieve fractal-analyse saboteert.
- **Oplossing:** Vervang de ZigZag-heuristiek volledig door een **Structuur-gestuurde Dynamic Swing Detection Engine**:
  1. **Ontkoppeling via Dependency Inversion:** Introduceer een abstracte interface `SwingDetector` of `PivotGenerator` in `pkg/model` zodat de API-laag (`api.go`) niet direct gekoppeld is aan een specifieke swing-heuristiek.
  2. **Raw Candle Swing Detection:** Genereer kandidaat-pivots puur op basis van lokale kaars-extrema (peaks/troughs) om alle potentiële trendbreuken te behouden.
  3. **Volatiliteit-Adaptieve Filtering:** Gebruik markt-adaptieve indicatoren zoals ATR (Average True Range) of Standard Deviation om ruis dynamisch te dempen, in plaats van vaste percentages.
  4. **Structuur-gestuurde Filtering (Fractal Validation):** Laat de uiteindelijke pivot-selectie valideren door de Elliott Wave regels zelf. Een pivot is pas valide als deze deel uitmaakt van een wiskundig sluitende sub-golf (bijv. Wave 2 retracement limit, Wave 3 length constraint). Dit garandeert een zuivere, theoretische opbouw van de Fractal Tree.

### [x] Stap 2: Recursieve, Gêneste Go-Modellen (`pkg/model/`)
- **Probleem:** De huidige Go-structs en JSON-contracts zijn volledig plat en lineair. Ze kunnen geen boomstructuur vasthouden.
- **Oplossing:** Wijzig `response.go` en `motive_wave.go` / `corrective_wave.go`. Introduceer een recursieve, zelfverwijzende array-structuur:
  ```go
  type WaveStructure struct {
      Type            string          `json:"type"`             // bijv. "MOTIVE_IMPULSE", "CORRECTIVE_ZIGZAG"
      Degree          string          `json:"degree"`           // "MINOR", "MINUTE", etc.
      ConfidenceScore float64         `json:"confidence_score"`
      Pivots          []Pivot         `json:"pivots"`           // De ankerpunten van deze specifieke golf
      SubStructures   []WaveStructure `json:"sub_structures"`   // NEW: De interne geneste sub-golven (Fractal Tree Children)
      PurpleBoxes     []TargetBox     `json:"purple_boxes"`
  }
  ```

### [x] Stap 3: Visualisatie & Future Purple Boxes Rendering (`frontend/src/components/ChartWidget.vue` & `BoxPrimitive.ts`)
- **Probleem 1 (Visualisatie):** Doordat geneste child-golven direct over de segmenten van hun parent-golven heen worden getekend, ontstaat er een optische verwarring waarbij de lijnen in elkaar overlopen. We willen ze visueel scheiden.
- **Probleem 2 (Purple Boxes):** Prognose-rechthoeken (Purple Boxes/Target Boxes) die in de toekomst liggen worden momenteel niet getekend. Dit komt doordat `timeScale.timeToCoordinate(time)` `null` retourneert voor tijdstippen in de toekomst waar nog geen kaarsen (candles) voor bestaan.
- **Oplossing:**
  1. **Visualisatie-afscheiding:** Geef geneste sub-golven een iets dunnere lijn (`lineWidth: 1.5` of `2` in plaats van `3`) en introduceer een subtiele, fractale stippellijn-visualisatie of kleurverzachting (bijvoorbeeld 80% opacity) om ze te onderscheiden van de hoofdtrend.
  2. **Future Purple Boxes Extrapolatie:** Geef de kaarsen-lijst (`candles`) door aan `BoxPrimitive`. Wanneer `timeScale.timeToCoordinate(time)` `null` retourneert voor start- of eindtijdstippen in de toekomst, bereken de coördinaat dan via lineaire extrapolatie op basis van de breedte (barSpacing) van de laatste twee bekende kaarsen en de tijdstap.