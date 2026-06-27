<template>
  <li class="tree-node">
    <button
      type="button"
      :class="['tree-row', { selected: node.id === selectedNode }]"
      @click="$emit('select', node.id)"
    >
      <span class="tree-toggle" @click.stop="expanded = !expanded">{{ hasChildren ? (expanded ? '−' : '+') : '·' }}</span>
      <span class="tree-degree">{{ node.degree.replaceAll('_', ' ') }}</span>
      <strong>{{ node.label || node.pattern.replaceAll('_', ' ') }}</strong>
      <span :class="['tree-state', node.status.toLowerCase()]">{{ node.status }}</span>
    </button>
    <ul v-if="expanded && hasChildren">
      <WaveTreeNode
        v-for="child in children"
        :key="child.id"
        :node="child"
        :nodes="nodes"
        :selected-node="selectedNode"
        @select="$emit('select', $event)"
      />
    </ul>
  </li>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MasterWaveNode } from '../types/api'

defineOptions({ name: 'WaveTreeNode' })
const props = defineProps<{
  node: MasterWaveNode
  nodes: MasterWaveNode[]
  selectedNode: string
}>()
defineEmits<{ select: [id: string] }>()
const expanded = ref(true)
const nodeByID = computed(() => new Map(props.nodes.map((node) => [node.id, node])))
const children = computed(() => props.node.child_ids
  .map((id) => nodeByID.value.get(id))
  .filter((node): node is MasterWaveNode => node !== undefined))
const hasChildren = computed(() => children.value.length > 0)
</script>
