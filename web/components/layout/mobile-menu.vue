<script setup lang="ts">
import { HomeIcon, TrophyIcon, UserIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

// Data-driven menu items
const menuItems = [
  {
    name: 'Home',
    icon: HomeIcon,
    path: '/home',
  },
  {
    name: 'Leaderboard',
    icon: TrophyIcon,
    path: '/leaderboard',
  },
  {
    name: 'Profile',
    icon: UserIcon,
    path: '/profile',
  },
]

// Get current route to highlight active item
const route = useRoute()
const currentPath = computed(() => route.path)
</script>

<template>
  <div class="fixed bottom-4 left-4 right-4 pb-2-safe flex items-center justify-center">
    <Card class="max-w-md rounded-full z-50 shadow bg-background border border-border">
      <nav class="mx-auto">
        <ul class="flex items-center h-12">
          <li v-for="item in menuItems" :key="item.name" class="flex flex-1 items-center">
            <NuxtLink
              :to="item.path"
              class="flex flex-col items-center py-2 w-20"
              :class="[
                currentPath === item.path
                  ? 'text-primary'
                  : 'text-muted-foreground hover:text-primary transition-colors',
              ]">
              <component :is="item.icon" class="h-5 w-5" />
            </NuxtLink>
          </li>
        </ul>
      </nav>
    </Card>
  </div>
</template>
