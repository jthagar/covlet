<script>
	import Sidebar from './Sidebar.svelte';
</script>

<main class="flex w-screen bg-black text-white">
	<div class="flex-[0.20] h-[90vh] overflow-y-hidden overflow-x-hidden gap-8">
		<div class="flex flex-col pl-1 pt-4 md:ml-1 lg:ml-3 xl:ml-5 w-[75%] gap-4">
			<Sidebar />
		</div>
	</div>
	<div class="flex-[0.80] height-[90vh] overflow-y-scroll bg-gray-600">
		<div class="flex flex-col pl-4 pt-4 md:ml-2 lg:ml-5 xl:ml-10 xl:mt-2 w-[90%] h-[90%] gap-4 bg-black text-white">
			<!-- Main content-->
			<h1>Combining SvelteKit and Golang server</h1>
		</div>
	</div>
</main>