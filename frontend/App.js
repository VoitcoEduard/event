import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import MainScreen from './MainScreen';
import AdminScreen from './AdminScreen';
import EventScreen from './EventScreen';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('main');
  const [currentEvent, setCurrentEvent] = useState(null);

  const handleJoinEvent = (event) => {
    setCurrentEvent(event);
    setCurrentScreen('event');
  };

  const renderScreen = () => {
    switch (currentScreen) {
      case 'admin':
        return <AdminScreen onNavigate={setCurrentScreen} />;
      case 'event':
        return <EventScreen event={currentEvent} onNavigate={setCurrentScreen} />;
      case 'main':
      default:
        return <MainScreen onNavigate={setCurrentScreen} onJoinEvent={handleJoinEvent} />;
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
