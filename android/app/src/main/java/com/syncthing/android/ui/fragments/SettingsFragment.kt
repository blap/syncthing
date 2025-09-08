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
import com.syncthing.android.R
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.viewmodel.MainViewModel
import com.syncthing.android.viewmodel.SyncthingViewModelFactory
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import com.google.gson.GsonBuilder

class SettingsFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var settingsText: TextView
    private lateinit var configOptionsText: TextView
    private lateinit var restartButton: Button
    private lateinit var shutdownButton: Button
    
    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View? {
        val view = inflater.inflate(R.layout.fragment_settings, container, false)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create(GsonBuilder().setLenient().create()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository)
        
        viewModel = ViewModelProvider(requireActivity(), factory)[MainViewModel::class.java]
        
        settingsText = view.findViewById(R.id.text_settings)
        configOptionsText = view.findViewById(R.id.text_config_options)
        restartButton = view.findViewById(R.id.button_restart)
        shutdownButton = view.findViewById(R.id.button_shutdown)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        restartButton.setOnClickListener {
            viewModel.restart()
            Toast.makeText(context, "Restarting Syncthing...", Toast.LENGTH_SHORT).show()
        }
        
        shutdownButton.setOnClickListener {
            viewModel.shutdown()
            Toast.makeText(context, "Shutting down Syncthing...", Toast.LENGTH_SHORT).show()
        }
        
        // Load initial data
        viewModel.fetchConfig()
        viewModel.fetchConfigOptions()
        observeData()
    }
    
    private fun observeData() {
        viewModel.config.observe(viewLifecycleOwner) { config ->
            settingsText.text = """
                Configuration Details:
                Version: ${config.version}
                Folders: ${config.folders.size}
                Devices: ${config.devices.size}
                GUI Address: ${config.gui.address}
                GUI Theme: ${config.gui.theme}
                GUI Enabled: ${config.gui.enabled}
                TLS Enabled: ${config.gui.useTLS}
                Debugging: ${config.gui.debugging}
                
                Options:
                Listen Addresses: ${config.options.listenAddresses.joinToString(", ")}
                Global Announce: ${config.options.globalAnnounceEnabled}
                Local Announce: ${config.options.localAnnounceEnabled}
                NAT Enabled: ${config.options.natEnabled}
                Relays Enabled: ${config.options.relaysEnabled}
                Auto Upgrade Interval: ${config.options.autoUpgradeIntervalH} hours
            """.trimIndent()
        }
        
        viewModel.configOptions.observe(viewLifecycleOwner) { options ->
            val formattedOptions = buildString {
                append("Global Options (${options.size} settings):\n")
                options.forEach { (key, value) ->
                    append("\n- $key: $value")
                }
            }
            configOptionsText.text = formattedOptions
        }
        
        viewModel.error.observe(viewLifecycleOwner) { error ->
            settingsText.text = "Error: $error"
            configOptionsText.text = "Error loading config options: $error"
        }
    }
}