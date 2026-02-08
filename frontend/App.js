import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import MainScreen from './MainScreen';
import AdminScreen from './AdminScreen';
import AdminEventDetailsScreen from './AdminEventDetailsScreen';
import EventScreen from './EventScreen';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('main');
  const [currentEvent, setCurrentEvent] = useState(null);

  React.useEffect(() => {
    if (typeof window !== 'undefined' && window.localStorage) {
      const savedScreen = localStorage.getItem('currentScreen');
      if (savedScreen) {
        if (savedScreen === 'event') {
          setCurrentScreen('main');
        } else {
          setCurrentScreen(savedScreen);
        }
      }

      // User ID Generation
      let uid = localStorage.getItem('userId');
      if (!uid) {
        uid = 'user_' + Math.random().toString(36).substr(2, 9);
        localStorage.setItem('userId', uid);
      }
    }
  }, []);

  const getUserId = () => {
    if (typeof window !== 'undefined' && window.localStorage) {
      return localStorage.getItem('userId');
    }
    return 'unknown_user';
  };

  const handleNavigate = (screen) => {
    setCurrentScreen(screen);
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.setItem('currentScreen', screen);
    }
  };

  const handleJoinEvent = (event) => {
    setCurrentEvent(event);
    handleNavigate('event');
  };

  const renderScreen = () => {
    switch (currentScreen) {
      case 'admin':
        return <AdminScreen onNavigate={handleNavigate} setEvent={setCurrentEvent} />;
      case 'admin_event_details': // New case
        return (
          <AdminEventDetailsScreen
            event={currentEvent}
            onNavigate={handleNavigate}
            onBack={() => handleNavigate('admin')}
          />
        );
      case 'event':
        return <EventScreen event={currentEvent} onNavigate={handleNavigate} userId={getUserId()} />;
      case 'main':
      default:
        return <MainScreen onNavigate={handleNavigate} onJoinEvent={handleJoinEvent} />;
    }
  };

  return (
    <View style={styles.container}>
      {renderScreen()}
      <StatusBar style="auto" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
});
