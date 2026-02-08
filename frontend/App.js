import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import MainScreen from './MainScreen';
import AdminScreen from './AdminScreen';
import EventScreen from './EventScreen';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('main');
  const [currentEvent, setCurrentEvent] = useState(null);

  React.useEffect(() => {
    if (typeof window !== 'undefined' && window.localStorage) {
      const savedScreen = localStorage.getItem('currentScreen');
      if (savedScreen) {
        if (savedScreen === 'event') {
          // If we were on event page, we need event data. 
          // For now, let's just go back to main if no event data is persisted.
          // Or we could persist event data too. 
          // Since the user only asked for admin page specifically:
          setCurrentScreen('main');
        } else {
          setCurrentScreen(savedScreen);
        }
      }
    }
  }, []);

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
        return <AdminScreen onNavigate={handleNavigate} />;
      case 'event':
        return <EventScreen event={currentEvent} onNavigate={handleNavigate} />;
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
