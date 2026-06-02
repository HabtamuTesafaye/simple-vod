<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 font-sans p-8">
    <div class="max-w-4xl mx-auto space-y-8">
      <h1 class="text-3xl font-bold text-gray-800 tracking-tight">VOD Admin Dashboard</h1>
      
      <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
        <!-- Upload Card -->
        <div class="bg-white p-6 rounded-xl shadow-sm border border-gray-100">
          <h2 class="text-xl font-semibold mb-4 text-gray-700">Upload Video</h2>
          <form @submit.prevent="uploadVideo" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-600 mb-1">Title</label>
              <input v-model="uploadData.title" required type="text" class="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none" placeholder="Video Title" />
            </div>
            
            <div>
              <label class="block text-sm font-medium text-gray-600 mb-1">Folder</label>
              <select v-model="uploadData.folder_id" class="w-full px-4 py-2 border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none">
                <option value="">Uncategorized</option>
                <option v-for="f in folders" :key="f.id" :value="f.id">{{ f.name }}</option>
              </select>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-600 mb-1">File</label>
              <input type="file" ref="fileInput" accept="video/mp4,video/quicktime" required class="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-semibold file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100" />
            </div>

            <button type="submit" :disabled="uploading" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-lg transition disabled:opacity-50">
              {{ uploading ? 'Uploading...' : 'Upload Video' }}
            </button>
            <p v-if="uploadStatus" class="text-sm mt-2" :class="uploadError ? 'text-red-500' : 'text-green-600'">{{ uploadStatus }}</p>
          </form>
        </div>

        <!-- Folders Card -->
        <div class="bg-white p-6 rounded-xl shadow-sm border border-gray-100 flex flex-col">
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-semibold text-gray-700">Folders</h2>
            <button @click="createFolder" class="bg-gray-100 hover:bg-gray-200 text-gray-700 text-sm font-medium py-1 px-3 rounded-md transition">+ New</button>
          </div>
          
          <ul class="space-y-2 flex-grow overflow-y-auto">
            <li @click="selectFolder('')" :class="selectedFolder === '' ? 'ring-2 ring-blue-500 bg-blue-50' : 'bg-gray-50 border-gray-100'" class="cursor-pointer flex justify-between items-center p-3 rounded-lg border hover:bg-blue-50 transition">
              <span class="font-medium text-gray-700">Uncategorized (Root)</span>
            </li>
            <li v-for="f in folders" :key="f.id" @click="selectFolder(f.id)" :class="selectedFolder === f.id ? 'ring-2 ring-blue-500 bg-blue-50' : 'bg-gray-50 border-gray-100'" class="cursor-pointer flex justify-between items-center p-3 rounded-lg border hover:bg-blue-50 transition">
              <span class="font-medium text-gray-700">{{ f.name }}</span>
              <button @click.stop="deleteFolder(f.id)" class="text-red-500 hover:text-red-700 text-sm transition">Delete</button>
            </li>
          </ul>
        </div>
      </div>

      <!-- Videos Card -->
      <div class="bg-white p-6 rounded-xl shadow-sm border border-gray-100">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold text-gray-700">Videos Library</h2>
          <span class="text-sm text-gray-500 bg-gray-100 px-3 py-1 rounded-full">
            Viewing: {{ selectedFolder ? folders.find(f => f.id === selectedFolder)?.name : 'Uncategorized' }}
          </span>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr class="bg-gray-50 text-gray-500 text-sm border-b border-gray-100">
                <th class="py-3 px-4 font-medium">Title</th>
                <th class="py-3 px-4 font-medium">Size</th>
                <th class="py-3 px-4 font-medium">Folder</th>
                <th class="py-3 px-4 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="v in videos" :key="v.id" class="border-b border-gray-50 hover:bg-gray-50/50 transition">
                <td class="py-3 px-4 font-medium text-gray-800">{{ v.title }}</td>
                <td class="py-3 px-4 text-sm text-gray-500">{{ (v.size_bytes / (1024*1024)).toFixed(2) }} MB</td>
                <td class="py-3 px-4 text-sm text-gray-500">
                  <span v-if="v.folder_id" class="px-2 py-1 bg-gray-100 rounded-md font-mono text-xs">{{ v.folder_id.substring(0,8) }}</span>
                  <span v-else class="text-gray-400 italic">Uncategorized</span>
                </td>
                <td class="py-3 px-4 text-right space-x-2">
                  <button @click="copyEmbed(v.id)" class="inline-block bg-blue-50 hover:bg-blue-100 text-blue-700 text-sm font-medium py-1 px-3 rounded-md transition">Embed</button>
                  <a :href="`http://localhost:8080/embed/${v.id}`" target="_blank" class="inline-block bg-emerald-100 hover:bg-emerald-200 text-emerald-700 text-sm font-medium py-1 px-3 rounded-md transition">Play</a>
                  <button @click="deleteVideo(v.id)" class="bg-red-50 hover:bg-red-100 text-red-600 text-sm font-medium py-1 px-3 rounded-md transition">Delete</button>
                </td>
              </tr>
              <tr v-if="videos.length === 0">
                <td colspan="4" class="py-8 text-center text-gray-500 italic">No videos uploaded yet</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const folders = ref([])
const videos = ref([])

const uploadData = ref({
  title: '',
  folder_id: ''
})
const fileInput = ref(null)
const uploading = ref(false)
const uploadStatus = ref('')
const uploadError = ref(false)
const selectedFolder = ref('')

async function selectFolder(id) {
  selectedFolder.value = id
  await fetchVideos()
}

async function fetchFolders() {
  try {
    const res = await fetch('/api/v1/folders')
    folders.value = await res.json()
  } catch (e) {
    console.error(e)
  }
}

async function fetchVideos() {
  try {
    const url = selectedFolder.value ? `/api/v1/videos?folder_id=${selectedFolder.value}` : '/api/v1/videos'
    const res = await fetch(url)
    const data = await res.json()
    videos.value = data.data || []
  } catch (e) {
    console.error(e)
  }
}

async function createFolder() {
  const name = prompt('Enter folder name:')
  if (!name) return
  
  await fetch('/api/v1/folders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name })
  })
  await fetchFolders()
}

async function deleteFolder(id) {
  if (!confirm('Are you sure you want to delete this folder?')) return
  await fetch(`/api/v1/folders/${id}`, { method: 'DELETE' })
  await fetchFolders()
}

async function deleteVideo(id) {
  if (!confirm('Are you sure you want to delete this video?')) return
  await fetch(`/api/v1/videos/${id}`, { method: 'DELETE' })
  await fetchVideos()
}

function copyEmbed(id) {
  const code = `<iframe src="http://localhost:8080/embed/${id}" width="640" height="360" frameborder="0" allow="autoplay; fullscreen" allowfullscreen></iframe>`
  navigator.clipboard.writeText(code)
  alert('Embed code copied to clipboard!\n\n' + code)
}

async function uploadVideo() {
  const file = fileInput.value.files[0]
  if (!file) return

  uploading.value = true
  uploadStatus.value = 'Uploading...'
  uploadError.value = false

  const formData = new FormData()
  formData.append('title', uploadData.value.title)
  formData.append('folder_id', uploadData.value.folder_id)
  formData.append('file', file)

  try {
    const res = await fetch('/api/v1/videos/upload', {
      method: 'POST',
      body: formData
    })
    
    if (res.ok) {
      uploadStatus.value = 'Upload complete!'
      uploadData.value.title = ''
      uploadData.value.folder_id = ''
      fileInput.value.value = ''
      await fetchVideos()
    } else {
      uploadError.value = true
      uploadStatus.value = `Error: ${await res.text()}`
    }
  } catch (e) {
    uploadError.value = true
    uploadStatus.value = `Error: ${e.message}`
  } finally {
    uploading.value = false
  }
}

onMounted(() => {
  fetchFolders()
  fetchVideos()
})
</script>
