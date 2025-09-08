package com.syncthing.android.ui.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import com.syncthing.android.R
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.viewmodel.MainViewModel
import com.syncthing.android.viewmodel.SyncthingViewModelFactory
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import com.google.gson.GsonBuilder

class FoldersFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var foldersText: TextView
    private lateinit var folderStatsText: TextView
    private lateinit var folderNeedText: TextView
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        val view = inflater.inflate(R.layout.fragment_folders, container, false)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create(GsonBuilder().setLenient().create()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository)
        
        viewModel = ViewModelProvider(requireActivity(), factory)[MainViewModel::class.java]
        
        foldersText = view.findViewById(R.id.text_folders)
        folderStatsText = view.findViewById(R.id.text_folder_stats)
        folderNeedText = view.findViewById(R.id.text_folder_need)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        loadData()
        observeData()
    }
    
    private fun loadData() {
        // Load folders data
        viewModel.fetchConfigFolders()
        viewModel.fetchFolderStats()
        // Note: folder need would be loaded for specific folders
        // This is just an example of how it would work
    }
    
    private fun observeData() {
        viewModel.configFolders.observe(viewLifecycleOwner) { folders ->
            val formattedFolders = buildString {
                append("Folders (${folders.size}):\n")
                folders.forEachIndexed { index, folder ->
                    append("\n${index + 1}. ${folder["label"]} (ID: ${folder["id"]})\n")
                    append("   Path: ${folder["path"]}\n")
                    append("   Type: ${folder["type"]}\n")
                    val devices = folder["devices"] as? List<*>
                    if (devices != null) {
                        append("   Devices: ${devices.size}\n")
                        devices.forEach { device ->
                            if (device is Map<*, *>) {
                                append("     - ${device["deviceID"]}\n")
                            }
                        }
                    }
                    append("   Rescan Interval: ${folder["rescanIntervalS"]}s\n")
                    append("   Paused: ${folder["paused"]}\n")
                }
            }
            foldersText.text = formattedFolders
        }
        
        viewModel.folderStats.observe(viewLifecycleOwner) { stats ->
            val formattedStats = buildString {
                append("Folder Statistics (${stats.size} folders):\n")
                stats.forEach { (folderId, folderData) ->
                    append("\n- Folder $folderId:\n")
                    if (folderData is Map<*, *>) {
                        append("  State: ${folderData["state"]}\n")
                        append("  In: ${formatBytes(folderData["inBytesTotal"] as? Long ?: 0)}\n")
                        append("  Out: ${formatBytes(folderData["outBytesTotal"] as? Long ?: 0)}\n")
                        append("  Need Files: ${folderData["needFiles"]}\n")
                        append("  Need Directories: ${folderData["needDirectories"]}\n")
                        append("  Need Symlinks: ${folderData["needSymlinks"]}\n")
                        append("  Need Deletes: ${folderData["needDeletes"]}\n")
                        append("  Need Bytes: ${formatBytes(folderData["needBytes"] as? Long ?: 0)}\n")
                        append("  Pull Errors: ${folderData["pullErrors"]}\n")
                        val lastFile = folderData["lastFile"] as? Map<*, *>
                        if (lastFile != null) {
                            append("  Last File: ${lastFile["filename"]} (${lastFile["at"]})\n")
                            append("  Action: ${lastFile["action"]}\n")
                        }
                    }
                }
            }
            folderStatsText.text = formattedStats
        }
        
        viewModel.folderNeedResults.observe(viewLifecycleOwner) { needResults ->
            val formattedNeed = buildString {
                append("Folder Need Data (${needResults.size} folders):\n")
                needResults.forEach { (folderId, needData) ->
                    append("\n- Folder $folderId:\n")
                    // needData is already Map<String, Any>, so no need for type check
                    append("  Total Items: ${needData["total"]}\n")
                    append("  Page: ${needData["page"]}/${needData["perpage"]}\n")
                    
                    val progress = needData["progress"] as? List<*>
                    if (progress != null && progress.isNotEmpty()) {
                        append("  Progress Items (${progress.size}):\n")
                        progress.take(3).forEach { item ->
                            if (item is Map<*, *>) {
                                append("    - ${item["name"]}: ${String.format("%.1f", item["completion"])}%\n")
                            }
                        }
                        if (progress.size > 3) {
                            append("    ... and ${progress.size - 3} more\n")
                        }
                    }
                    
                    val queued = needData["queued"] as? List<*>
                    if (queued != null && queued.isNotEmpty()) {
                        append("  Queued Items (${queued.size}):\n")
                        queued.take(3).forEach { item ->
                            if (item is Map<*, *>) {
                                append("    - ${item["name"]}: ${formatBytes(item["size"] as? Long ?: 0)}\n")
                            }
                        }
                        if (queued.size > 3) {
                            append("    ... and ${queued.size - 3} more\n")
                        }
                    }
                }
            }
            folderNeedText.text = formattedNeed
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            foldersText.text = "Error: $error"
        }
    }
    
    private fun formatBytes(bytes: Long): String {
        return when {
            bytes >= 1024 * 1024 * 1024 -> String.format("%.2f GB", bytes / (1024.0 * 1024.0 * 1024.0))
            bytes >= 1024 * 1024 -> String.format("%.2f MB", bytes / (1024.0 * 1024.0))
            bytes >= 1024 -> String.format("%.2f KB", bytes / 1024.0)
            else -> "$bytes B"
        }
    }
}