<script>
	import Sidebar from './Sidebar.svelte';
</script>

<main class="flex w-screen bg-black text-white">
	<div class="flex-[0.21] h-[90vh] overflow-y-hidden overflow-x-hidden gap-">
		<div class="flex flex-col w-[90%] gap-4">
			<Sidebar />
		</div>
	</div>
	<div class="flex-[0.80] height-[90vh] overflow-y-scroll bg-gray-600">
		<div class="flex flex-col pl-4 pt-4 md:ml-2 lg:ml-5 xl:ml-10 xl:mt-2 w-[100%] h-[100%] gap-4 bg-black text-white">
			<!-- Main content-->
			<h1>Combining SvelteKit and Golang server</h1>
		</div>
	</div>
</main>