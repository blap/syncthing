package com.syncthing.android.ui.fragments

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import android.app.Application
import com.syncthing.android.R
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.viewmodel.MainViewModel
import com.syncthing.android.viewmodel.SyncthingViewModelFactory
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import com.google.gson.Gson
import android.content.Context

class ConfigurationFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var foldersInfoText: TextView
    private lateinit var devicesInfoText: TextView
    private lateinit var optionsInfoText: TextView
    private lateinit var guiInfoText: TextView
    private lateinit var manageFoldersButton: Button
    private lateinit var manageDevicesButton: Button
    private lateinit var manageOptionsButton: Button
    private lateinit var manageGuiButton: Button
    private lateinit var restartButton: Button
    private lateinit var shutdownButton: Button
    
    // API key - in a real implementation, this would come from secure storage
    private val apiKey = "YOUR_API_KEY_HERE"
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        val view = inflater.inflate(R.layout.fragment_configuration, container, false)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create(Gson()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository, requireActivity().application)
        
        viewModel = ViewModelProvider(this, factory)[MainViewModel::class.java]
        
        // Initialize views
        foldersInfoText = view.findViewById(R.id.text_folders_info)
        devicesInfoText = view.findViewById(R.id.text_devices_info)
        optionsInfoText = view.findViewById(R.id.text_options_info)
        guiInfoText = view.findViewById(R.id.text_gui_info)
        manageFoldersButton = view.findViewById(R.id.btn_manage_folders)
        manageDevicesButton = view.findViewById(R.id.btn_manage_devices)
        manageOptionsButton = view.findViewById(R.id.btn_manage_options)
        manageGuiButton = view.findViewById(R.id.btn_manage_gui)
        restartButton = view.findViewById(R.id.btn_restart)
        shutdownButton = view.findViewById(R.id.btn_shutdown)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        setupUI()
        observeData()
        loadData()
    }
    
    private fun setupUI() {
        manageFoldersButton.setOnClickListener {
            // In a real implementation, this would navigate to a folders management screen
            Toast.makeText(context, "Manage Folders clicked", Toast.LENGTH_SHORT).show()
        }
        
        manageDevicesButton.setOnClickListener {
            // In a real implementation, this would navigate to a devices management screen
            Toast.makeText(context, "Manage Devices clicked", Toast.LENGTH_SHORT).show()
        }
        
        manageOptionsButton.setOnClickListener {
            // In a real implementation, this would navigate to an options management screen
            Toast.makeText(context, "Manage Options clicked", Toast.LENGTH_SHORT).show()
        }
        
        manageGuiButton.setOnClickListener {
            // In a real implementation, this would navigate to a GUI settings management screen
            Toast.makeText(context, "Manage GUI Settings clicked", Toast.LENGTH_SHORT).show()
        }
        
        restartButton.setOnClickListener {
            restartSystem()
        }
        
        shutdownButton.setOnClickListener {
            shutdownSystem()
        }
    }
    
    private fun observeData() {
        viewModel.folders.observe(viewLifecycleOwner) { folders ->
            foldersInfoText.text = "Folders: ${folders.size} configured\nClick to manage folder settings"
        }
        
        viewModel.devices.observe(viewLifecycleOwner) { devices ->
            devicesInfoText.text = "Devices: ${devices.size} configured\nClick to manage device settings"
        }
        
        viewModel.options.observe(viewLifecycleOwner) { options ->
            optionsInfoText.text = "Options: Global settings\nClick to manage global options"
        }
        
        viewModel.guiSettings.observe(viewLifecycleOwner) { guiSettings ->
            guiInfoText.text = "GUI Settings: Theme and interface\nClick to manage GUI settings"
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            if (isLoading) {
                foldersInfoText.text = "Folders: Loading..."
                devicesInfoText.text = "Devices: Loading..."
                optionsInfoText.text = "Options: Loading..."
                guiInfoText.text = "GUI Settings: Loading..."
            }
        }
        
        viewModel.error.observe(viewLifecycleOwner) { errorMsg ->
            if (errorMsg != null) {
                Toast.makeText(context, "Error: $errorMsg", Toast.LENGTH_LONG).show()
            }
        }
    }
    
    private fun loadData() {
        // Load configuration data
        viewModel.fetchFolders(apiKey)
        viewModel.fetchDevices(apiKey)
        viewModel.fetchOptions(apiKey)
        viewModel.fetchGuiSettings(apiKey)
    }
    
    private fun restartSystem() {
        viewModel.restartSystem(apiKey)
    }
    
    private fun shutdownSystem() {
        viewModel.shutdownSystem(apiKey)
    }
}