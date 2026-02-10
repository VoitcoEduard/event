import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ActivityIndicator, ScrollView } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { Ionicons } from '@expo/vector-icons';
import FormModule from './components/modules/FormModule';

// Use environment variable or default to localhost for development
const API_URL = typeof window !== 'undefined' && window.location
    ? (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
        ? 'http://localhost:8080'
        : `http://${window.location.hostname}:8080`)
    : 'http://localhost:8080';

export default function EventScreen({ event, onNavigate, userId }) {
    const [currentEvent, setCurrentEvent] = useState(event);
    const [activeModules, setActiveModules] = useState([]);
    const [selectedModule, setSelectedModule] = useState(null);

    useEffect(() => {
        if (!event?.id) return;

        // Poll for event updates every 3 seconds
        const interval = setInterval(async () => {
            try {
                const response = await fetch(`${API_URL}/events/${event.id}`);
                if (response.ok) {
                    const data = await response.json();
                    setCurrentEvent(data);

                    // Check for active modules
                    if (data.modules) {
                        const active = data.modules.filter(m => m.isActive);
                        setActiveModules(active);
                    }
                }
            } catch (error) {
                console.error('Polling error:', error);
            }
        }, 3000);

        return () => clearInterval(interval);
    }, [event?.id]);

    const handleModuleComplete = () => {
        // Optionally remove from list or just show completed state within module
        // For now, we keep it selected but FormModule shows "Thank you"
        // Or we can deselect after a delay
        setTimeout(() => {
            setSelectedModule(null);
        }, 2000);
    };

    if (!currentEvent) {
        return (
            <View style={styles.container}>
                <Text style={styles.errorText}>No event data found.</Text>
                <TouchableOpacity onPress={() => onNavigate('main')} style={styles.backButton}>
                    <Text style={styles.backButtonText}>Go Back</Text>
                </TouchableOpacity>
            </View>
        );
    }

    if (selectedModule) {
        return (
            <View style={styles.container}>
                <StatusBar style="light" />
                <View style={styles.header}>
                    <TouchableOpacity onPress={() => setSelectedModule(null)} style={styles.backLink}>
                        <Ionicons name="arrow-back" size={24} color="#FFF" />
                        <Text style={styles.backLinkText}>Back to Event</Text>
                    </TouchableOpacity>
                </View>
                <View style={styles.moduleWrapper}>
                    {selectedModule.type === 'form' && (
                        <FormModule
                            module={selectedModule}
                            eventId={currentEvent.id}
                            userId={userId}
                            onComplete={handleModuleComplete}
                        />
                    )}
                </View>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <StatusBar style="light" />
            <View style={styles.content}>
                <View style={styles.header}>
                    <Text style={styles.statusText}>Connected to</Text>
                    <Text style={styles.eventName}>{currentEvent.name}</Text>
                </View>

                <View style={styles.listContainer}>
                    {activeModules.length > 0 ? (
                        <>
                            <Text style={styles.sectionTitle}>Active Activities</Text>
                            {activeModules.map(module => (
                                <TouchableOpacity
                                    key={module.id}
                                    style={styles.moduleCard}
                                    onPress={() => setSelectedModule(module)}
                                >
                                    <View style={styles.iconContainer}>
                                        <Text style={styles.moduleIcon}>
                                            {module.type === 'form' ? '📝' : '📊'}
                                        </Text>
                                    </View>
                                    <View style={styles.moduleInfo}>
                                        <Text style={styles.moduleTitle}>{module.name}</Text>
                                        <Text style={styles.moduleSubtitle}>Tap to start</Text>
                                    </View>
                                    <Ionicons name="chevron-forward" size={24} color="#666" />
                                </TouchableOpacity>
                            ))}
                        </>
                    ) : (
                        <View style={styles.waitingCard}>
                            <ActivityIndicator size="large" color="#007AFF" style={styles.loader} />
                            <Text style={styles.message}>Waiting for host to start the next activity...</Text>
                            <Text style={styles.subMessage}>Sit tight and get ready!</Text>
                        </View>
                    )}
                </View>

                <TouchableOpacity
                    style={styles.leaveButton}
                    onPress={() => onNavigate('main')}
                >
                    <Text style={styles.leaveButtonText}>Leave Event</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
        alignItems: 'center',
        justifyContent: 'center',
    },
    content: {
        width: '90%',
        maxWidth: 400,
        alignItems: 'center',
        flex: 1,
        paddingTop: 60,
    },
    header: {
        marginBottom: 40,
        alignItems: 'center',
        width: '100%',
    },
    statusText: {
        color: '#888',
        fontSize: 14,
        textTransform: 'uppercase',
        letterSpacing: 1,
        marginBottom: 8,
    },
    eventName: {
        color: '#FFF',
        fontSize: 32,
        fontWeight: '800',
        textAlign: 'center',
    },
    listContainer: {
        width: '100%',
        flex: 1,
    },
    sectionTitle: {
        color: '#888',
        fontSize: 14,
        fontWeight: '600',
        marginBottom: 16,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    moduleCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 16,
        padding: 16,
        marginBottom: 16,
        flexDirection: 'row',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#333',
    },
    iconContainer: {
        width: 48,
        height: 48,
        borderRadius: 12,
        backgroundColor: '#333',
        alignItems: 'center',
        justifyContent: 'center',
        marginRight: 16,
    },
    moduleIcon: {
        fontSize: 24,
    },
    moduleInfo: {
        flex: 1,
    },
    moduleTitle: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: '700',
        marginBottom: 4,
    },
    moduleSubtitle: {
        color: '#30D158',
        fontSize: 14,
        fontWeight: '500',
    },
    waitingCard: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 40,
        width: '100%',
        alignItems: 'center',
        marginBottom: 40,
        borderWidth: 1,
        borderColor: '#333',
    },
    loader: {
        marginBottom: 20,
    },
    message: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: '600',
        textAlign: 'center',
        marginBottom: 8,
    },
    subMessage: {
        color: '#888',
        fontSize: 14,
        textAlign: 'center',
    },
    leaveButton: {
        paddingVertical: 16,
        paddingHorizontal: 32,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: '#333',
        width: '100%',
        alignItems: 'center',
        marginBottom: 40,
    },
    leaveButtonText: {
        color: '#FF453A',
        fontSize: 16,
        fontWeight: '600',
    },
    errorText: {
        color: '#FF453A',
        fontSize: 18,
        marginBottom: 20,
    },
    backButton: {
        padding: 10,
    },
    backButtonText: {
        color: '#FFF',
        fontSize: 16,
    },
    backLink: {
        flexDirection: 'row',
        alignItems: 'center',
        alignSelf: 'flex-start',
        padding: 16,
        marginBottom: 10,
    },
    backLinkText: {
        color: '#FFF',
        fontSize: 16,
        marginLeft: 8,
    },
    moduleWrapper: {
        flex: 1,
        width: '100%',
        maxWidth: 600,
        paddingHorizontal: 20,
    }
});
