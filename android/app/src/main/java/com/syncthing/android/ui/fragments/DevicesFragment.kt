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

class DevicesFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var devicesText: TextView
    private lateinit var deviceStatsText: TextView
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        val view = inflater.inflate(R.layout.fragment_devices, container, false)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create(GsonBuilder().setLenient().create()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository)
        
        viewModel = ViewModelProvider(requireActivity(), factory)[MainViewModel::class.java]
        
        devicesText = view.findViewById(R.id.text_devices)
        deviceStatsText = view.findViewById(R.id.text_device_stats)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        loadData()
        observeData()
    }
    
    private fun loadData() {
        // Load devices data
        viewModel.fetchConnections()
        viewModel.fetchDeviceStats()
        viewModel.fetchConfigDevices()
    }
    
    private fun observeData() {
        viewModel.connections.observe(viewLifecycleOwner) { connections ->
            val formattedConnections = buildString {
                append("Device Connections (${connections.size - 1} devices):\n") // -1 for "total"
                connections.forEach { (deviceId, connectionData) ->
                    if (deviceId != "total" && connectionData is Map<*, *>) {
                        append("\n- Device $deviceId:\n")
                        append("  Connected: ${connectionData["connected"]}\n")
                        append("  In: ${formatBytes(connectionData["inBytesTotal"] as? Long ?: 0)}\n")
                        append("  Out: ${formatBytes(connectionData["outBytesTotal"] as? Long ?: 0)}\n")
                        append("  Type: ${connectionData["type"]}\n")
                        val address = connectionData["address"] as? String
                        if (address != null) {
                            append("  Address: $address\n")
                        }
                        val clientVersion = connectionData["clientVersion"] as? String
                        if (clientVersion != null) {
                            append("  Client Version: $clientVersion\n")
                        }
                    }
                }
            }
            devicesText.text = formattedConnections
        }
        
        viewModel.deviceStats.observe(viewLifecycleOwner) { stats ->
            val formattedStats = buildString {
                append("Device Statistics (${stats.size} devices):\n")
                stats.forEach { (deviceId, deviceData) ->
                    append("\n- Device ${deviceId.take(8)}...:\n")
                    if (deviceData is Map<*, *>) {
                        append("  Last Seen: ${deviceData["lastSeen"]}\n")
                        append("  In: ${formatBytes(deviceData["inBytesTotal"] as? Long ?: 0)}\n")
                        append("  Out: ${formatBytes(deviceData["outBytesTotal"] as? Long ?: 0)}\n")
                        val lastConnection = deviceData["lastConnection"] as? String
                        if (lastConnection != null) {
                            append("  Last Connection: $lastConnection\n")
                        }
                    }
                }
            }
            deviceStatsText.text = formattedStats
        }
        
        viewModel.configDevices.observe(viewLifecycleOwner) { devices ->
            if (deviceStatsText.text.startsWith("Device Statistics")) {
                deviceStatsText.text = deviceStatsText.text.toString() + "\n\nConfig Devices: ${devices.size}"
            }
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            devicesText.text = "Error: $error"
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