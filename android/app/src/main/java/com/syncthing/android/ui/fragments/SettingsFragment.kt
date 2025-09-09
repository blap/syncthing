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
import com.google.gson.Gson

class SettingsFragment : Fragment() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var settingsText: TextView
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
            .addConverterFactory(GsonConverterFactory.create(Gson()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository, requireActivity().application)
        
        viewModel = ViewModelProvider(this, factory)[MainViewModel::class.java]
        
        settingsText = view.findViewById(R.id.text_settings)
        restartButton = view.findViewById(R.id.button_restart)
        shutdownButton = view.findViewById(R.id.button_shutdown)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        
        restartButton.setOnClickListener {
            Toast.makeText(context, "Restart functionality would be implemented here", Toast.LENGTH_SHORT).show()
        }
        
        shutdownButton.setOnClickListener {
            Toast.makeText(context, "Shutdown functionality would be implemented here", Toast.LENGTH_SHORT).show()
        }
        
        observeData()
    }
    
    private fun observeData() {
        viewModel.systemStatus.observe(viewLifecycleOwner) { status ->
            settingsText.text = """
                Settings
                Device ID: ${status.myID}
            """.trimIndent()
        }
        
        viewModel.systemVersion.observe(viewLifecycleOwner) { version ->
            settingsText.append("\nVersion: ${version.version}")
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            if (isLoading) {
                settingsText.text = "Loading..."
            }
        }
        
        viewModel.error.observe(viewLifecycleOwner) { errorMsg ->
            settingsText.text = "Error: ${errorMsg ?: "Unknown error"}"
        }
    }
}