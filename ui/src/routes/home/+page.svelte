<script>
    import Sidebar from '../Sidebar.svelte';
</script>

<main class="flex w-screen bg-black text-white">
    <div class="flex-[0.21] h-[90vh] overflow-y-hidden overflow-x-hidden gap-">
        <div class="flex flex-col w-[90%] gap-4">
            <Sidebar />
        </div>
    </div>
    <div class="flex-[0.80] height-[90vh] overflow-y-scroll">
        <div class="flex flex-col pt-8 md:ml-2 lg:ml-5 xl:ml-10 xl:mt-2 w-[100%] h-[100%] gap-4 bg-black text-white">
            <!-- Main content-->
            <div class="flex rounded-[2vw] pl-5 pt-5 h-[95%] w-[92%] md:ml-2 lg:ml-5 xl:ml-10 xl:mt-2 bg-gray-600/40 border-gray-600 border-1 text-white">
                <p class="text-shadow-lg/30">Combining SvelteKit and Golang server</p>
            </div>
        </div>
    </div>
</main>