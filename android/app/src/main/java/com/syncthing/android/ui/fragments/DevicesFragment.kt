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
            .addConverterFactory(GsonConverterFactory.create(Gson()))
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository, requireActivity().application)
        
        viewModel = ViewModelProvider(this, factory)[MainViewModel::class.java]
        
        devicesText = view.findViewById(R.id.text_devices)
        deviceStatsText = view.findViewById(R.id.text_device_stats)
        
        return view
    }
    
    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        observeData()
    }
    
    private fun observeData() {
        viewModel.systemStatus.observe(viewLifecycleOwner) { status ->
            devicesText.text = "Device ID: ${status.myID}"
        }
        
        viewModel.isLoading.observe(viewLifecycleOwner) { isLoading ->
            if (isLoading) {
                devicesText.text = "Loading..."
            }
        }
        
        viewModel.error.observe(viewLifecycleOwner) { errorMsg ->
            devicesText.text = "Error: ${errorMsg ?: "Unknown error"}"
        }
    }
}