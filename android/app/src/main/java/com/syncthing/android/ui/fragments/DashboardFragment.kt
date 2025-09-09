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
import com.google.gson.Gson

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
            .addConverterFactory(GsonConverterFactory.create(Gson()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository, requireActivity().application)
        
        viewModel = ViewModelProvider(this, factory)[MainViewModel::class.java]
        
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
        // We need an API key to fetch data, which would normally come from user input
        // For now, we'll just observe the data that might be loaded elsewhere
        observeData()
    }
    
    private fun setupUI() {
        // Setup dashboard UI components
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
        
        viewModel.systemVersion.observe(viewLifecycleOwner) { version ->
            systemStatusText.append("\n\nVersion: ${version.version}")
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            if (isLoading) {
                systemStatusText.text = "Loading..."
            }
        }
        
        viewModel.error.observe(viewLifecycleOwner) { errorMsg ->
            systemStatusText.text = "Error: ${errorMsg ?: "Unknown error"}"
        }
    }
    
    private fun formatUptime(seconds: Long): String {
        val hours = seconds / 3600
        val minutes = (seconds % 3600) / 60
        val secs = seconds % 60
        return "${hours}h ${minutes}m ${secs}s"
    }
}