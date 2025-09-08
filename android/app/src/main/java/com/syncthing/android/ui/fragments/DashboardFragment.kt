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

class DashboardFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var systemStatusText: TextView
    private lateinit var connectionsText: TextView
    private lateinit var eventsText: TextView
    private lateinit var deviceStatsText: TextView
    private lateinit var folderStatsText: TextView
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        val view = inflater.inflate(R.layout.fragment_dashboard, container, false)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create(GsonBuilder().setLenient().create()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository)
        
        viewModel = ViewModelProvider(requireActivity(), factory)[MainViewModel::class.java]
        
        systemStatusText = view.findViewById(R.id.text_system_status)
        connectionsText = view.findViewById(R.id.text_connections)
        eventsText = view.findViewById(R.id.text_events)
        deviceStatsText = view.findViewById(R.id.text_device_stats)
        folderStatsText = view.findViewById(R.id.text_folder_stats)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        setupUI()
        loadData()
        observeData()
    }
    
    private fun setupUI() {
        // Setup dashboard UI components
    }
    
    private fun loadData() {
        // Load initial data
        viewModel.fetchSystemStatus()
        viewModel.fetchConnections()
        viewModel.fetchEvents()
        viewModel.fetchDeviceStats()
        viewModel.fetchFolderStats()
    }
    
    private fun observeData() {
        viewModel.systemStatus.observe(viewLifecycleOwner) { status ->
            systemStatusText.text = """
                Device ID: ${status.myID}
                CPU Usage: ${String.format("%.2f", status.cpuPercent)}%
                Memory: ${status.alloc / (1024L * 1024L)} MB / ${status.sys / (1024L * 1024L)} MB
                Uptime: ${formatUptime(status.uptime.toLong())}
                Goroutines: ${status.goroutines}
                Discovery Methods: ${status.discoveryMethods}
                Path Separator: ${status.pathSeparator}
                Start Time: ${status.startTime}
            """.trimIndent()
        }
        
        viewModel.connections.observe(viewLifecycleOwner) { connections ->
            val total = connections["total"] as? Map<*, *>
            if (total != null) {
                connectionsText.text = """
                    Total Connections:
                    In: ${formatBytes(total["inBytesTotal"] as? Long ?: 0L)} bytes
                    Out: ${formatBytes(total["outBytesTotal"] as? Long ?: 0L)} bytes
                    Last Updated: ${total["at"]}
                """.trimIndent()
            } else {
                connectionsText.text = "No connection data available"
            }
        }
        
        viewModel.events.observe(viewLifecycleOwner) { events ->
            val recentEvents = events.take(5) // Show only the 5 most recent events
            eventsText.text = """
                Recent Events (${events.size} total):
                ${recentEvents.joinToString("\n") { event ->
                    "- ${event["type"]}: ${event["time"]}"
                }}
            """.trimIndent()
        }
        
        viewModel.deviceStats.observe(viewLifecycleOwner) { stats ->
            val formattedStats = buildString {
                append("Device Statistics (${stats.size} devices):\n")
                stats.forEach { (deviceId, deviceData) ->
                    append("\n- Device ${deviceId.take(8)}...:\n")
                    if (deviceData is Map<*, *>) {
                        append("  Last Seen: ${deviceData["lastSeen"]}\n")
                        append("  In: ${formatBytes(deviceData["inBytesTotal"] as? Long ?: 0L)}\n")
                        append("  Out: ${formatBytes(deviceData["outBytesTotal"] as? Long ?: 0L)}\n")
                        val lastConnection = deviceData["lastConnection"] as? String
                        if (lastConnection != null) {
                            append("  Last Connection: $lastConnection\n")
                        }
                    }
                }
            }
            deviceStatsText.text = formattedStats
        }
        
        viewModel.folderStats.observe(viewLifecycleOwner) { stats ->
            val formattedStats = buildString {
                append("Folder Statistics (${stats.size} folders):\n")
                stats.forEach { (folderId, folderData) ->
                    append("\n- Folder $folderId:\n")
                    if (folderData is Map<*, *>) {
                        append("  State: ${folderData["state"]}\n")
                        append("  In: ${formatBytes(folderData["inBytesTotal"] as? Long ?: 0L)}\n")
                        append("  Out: ${formatBytes(folderData["outBytesTotal"] as? Long ?: 0L)}\n")
                        append("  Need Files: ${folderData["needFiles"]}\n")
                        append("  Need Bytes: ${formatBytes(folderData["needBytes"] as? Long ?: 0L)}\n")
                        append("  Pull Errors: ${folderData["pullErrors"]}\n")
                        val lastFile = folderData["lastFile"] as? Map<*, *>
                        if (lastFile != null) {
                            append("  Last File: ${lastFile["filename"]} (${lastFile["at"]})\n")
                        }
                    }
                }
            }
            folderStatsText.text = formattedStats
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            systemStatusText.text = "Error: $error"
        }
    }
    
    private fun formatUptime(seconds: Long): String {
        val hours = seconds / 3600
        val minutes = (seconds % 3600) / 60
        val secs = seconds % 60
        return "${hours}h ${minutes}m ${secs}s"
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