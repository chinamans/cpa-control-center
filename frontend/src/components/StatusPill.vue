<script setup lang="ts">
import { computed } from 'vue'
import { ElTag } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { stateLabel, statusTagType } from '@/utils/status'

const { t } = useI18n()

const props = defineProps<{
  state: string
  label?: string
}>()

const tone = computed(() => statusTagType(props.state))
const displayLabel = computed(() => props.label || stateLabel(props.state) || t('common.unknown'))
</script>

<template>
  <ElTag :type="tone" effect="dark" round>
    {{ displayLabel }}
  </ElTag>
</template>
