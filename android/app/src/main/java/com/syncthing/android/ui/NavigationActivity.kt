package com.syncthing.android.ui

import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import androidx.fragment.app.Fragment
import androidx.lifecycle.ViewModelProvider
import com.google.android.material.bottomnavigation.BottomNavigationView
import com.syncthing.android.R
import com.syncthing.android.data.api.SyncthingApiServiceInterface
import com.syncthing.android.data.repository.SyncthingRepository
import com.syncthing.android.ui.fragments.DashboardFragment
import com.syncthing.android.ui.fragments.FoldersFragment
import com.syncthing.android.ui.fragments.DevicesFragment
import com.syncthing.android.ui.fragments.ConfigurationFragment
import com.syncthing.android.ui.fragments.SettingsFragment
import com.syncthing.android.viewmodel.MainViewModel
import com.syncthing.android.viewmodel.SyncthingViewModelFactory
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import android.content.Context
import android.view.MenuItem

class NavigationActivity : AppCompatActivity() {
    
    private lateinit var viewModel: MainViewModel
    private lateinit var bottomNavigation: BottomNavigationView
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_navigation)
        
        // Initialize Retrofit and dependencies
        val retrofit = Retrofit.Builder()
            .baseUrl("http://localhost:8384") // Default Syncthing API URL
            .addConverterFactory(GsonConverterFactory.create())
            .build()
        
        val apiService = retrofit.create(SyncthingApiServiceInterface::class.java)
        val repository = SyncthingRepository(apiService)
        val factory = SyncthingViewModelFactory(repository, application)
        
        viewModel = ViewModelProvider(this, factory)[MainViewModel::class.java]
        
        bottomNavigation = findViewById(R.id.bottom_navigation)
        bottomNavigation.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.navigation_dashboard -> {
                    loadFragment(DashboardFragment())
                    true
                }
                R.id.navigation_folders -> {
                    loadFragment(FoldersFragment())
                    true
                }
                R.id.navigation_devices -> {
                    loadFragment(DevicesFragment())
                    true
                }
                R.id.navigation_configuration -> {
                    loadFragment(ConfigurationFragment())
                    true
                }
                R.id.navigation_settings -> {
                    loadFragment(SettingsFragment())
                    true
                }
                else -> false
            }
        }
        
        // Load default fragment
        if (savedInstanceState == null) {
            loadFragment(DashboardFragment())
        }
    }
    
    private fun loadFragment(fragment: Fragment) {
        supportFragmentManager.beginTransaction()
            .replace(R.id.fragment_container, fragment)
            .commit()
    }
}