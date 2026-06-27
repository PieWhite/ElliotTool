<template>
  <li class="tree-node">
    <button type="button" class="tree-row" @click="expanded = !expanded">
      <span class="tree-toggle">{{ hasChildren ? (expanded ? '−' : '+') : '·' }}</span>
      <span class="tree-degree">{{ node.degree.replaceAll('_', ' ') }}</span>
      <strong>{{ node.label || node.pattern.replaceAll('_', ' ') }}</strong>
      <span :class="['tree-state', node.status.toLowerCase()]">{{ node.status }}</span>
    </button>
    <ul v-if="expanded && hasChildren">
      <WaveTreeNode v-for="child in node.children" :key="child.id" :node="child" />
    </ul>
  </li>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { WaveNode } from '../types/api'

defineOptions({ name: 'WaveTreeNode' })
const props = defineProps<{ node: WaveNode }>()
const expanded = ref(true)
const hasChildren = computed(() => (props.node.children?.length ?? 0) > 0)
</script>
