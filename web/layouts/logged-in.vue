<template>
  <!-- The Default Layout if no layout is specificed -->
  <div class="h-screen w-screen flex py-10 justify-center items-center">
    <slot />

    <LayoutMobileMenu />
  </div>
</template>
