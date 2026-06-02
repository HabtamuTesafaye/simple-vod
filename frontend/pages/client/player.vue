<template>
  <div class="h-screen bg-black flex items-center justify-center p-4">
    <div v-if="error" class="text-red-500 bg-red-500/10 p-6 rounded-lg font-mono">
      {{ error }}
    </div>
    
    <div v-else-if="loading" class="text-white/50 animate-pulse font-medium">
      Loading secure stream...
    </div>

    <div v-else class="w-full max-w-6xl w-full">
      <div class="bg-gray-900 rounded-xl overflow-hidden shadow-2xl ring-1 ring-white/10">
        <!-- Player Header -->
        <div class="px-6 py-4 border-b border-white/5 flex justify-between items-center bg-gray-900/50">
          <h2 class="text-lg font-semibold text-white tracking-wide">{{ videoData?.title }}</h2>
          <div class="text-xs text-emerald-400 font-mono bg-emerald-400/10 px-2 py-1 rounded">
            URL Token Active (Exp: {{ videoData?.expires_in }}s)
          </div>
        </div>
        
        <!-- Native Video Player -->
        <video 
          ref="videoPlayer"
          controls 
          playsinline 
          autoplay
          :src="videoData?.url"
          class="w-full aspect-video outline-none bg-black"
        >
          Your browser does not support the video tag.
        </video>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const videoId = route.query.id

const loading = ref(true)
const error = ref(null)
const videoData = ref(null)
const videoPlayer = ref(null)
let refreshTimer = null

async function loadStream() {
  if (!videoId) {
    error.value = "Missing video ID in URL (?id=...)"
    loading.value = false
    return
  }

  try {
    const res = await fetch(`/api/v1/videos/${videoId}/stream`)
    if (!res.ok) {
      throw new Error(await res.text())
    }
    
    videoData.value = await res.json()
    
    // Auto-refresh token at 75% of lifetime
    const refreshMs = videoData.value.expires_in * 0.75 * 1000
    refreshTimer = setTimeout(refreshStreamToken, refreshMs)
    
  } catch (e) {
    error.value = `Failed to load video: ${e.message}`
  } finally {
    loading.value = false
  }
}

async function refreshStreamToken() {
  try {
    const res = await fetch(`/api/v1/videos/${videoId}/stream`)
    if (res.ok) {
      const data = await res.json()
      videoData.value = data
      
      // Update source transparently without breaking playback
      if (videoPlayer.value) {
        const currentTime = videoPlayer.value.currentTime
        const isPaused = videoPlayer.value.paused
        
        videoPlayer.value.src = data.url
        videoPlayer.value.currentTime = currentTime
        if (!isPaused) videoPlayer.value.play()
      }
      
      refreshTimer = setTimeout(refreshStreamToken, data.expires_in * 0.75 * 1000)
    }
  } catch (e) {
    console.error("Failed to refresh token", e)
  }
}

onMounted(() => {
  loadStream()
})

onUnmounted(() => {
  if (refreshTimer) clearTimeout(refreshTimer)
})
</script>
