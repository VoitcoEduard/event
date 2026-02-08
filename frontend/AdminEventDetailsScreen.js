import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ScrollView, Animated, Clipboard, Alert, Platform } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import QRCode from 'react-qr-code';

const API_URL = 'http://localhost:8080';

export default function AdminEventDetailsScreen({ event, onNavigate, onBack }) {
    const [currentEvent, setCurrentEvent] = useState(event);
    const [activeStatusAnim] = useState(new Animated.Value(event?.isActive ? 1 : 0));

    useEffect(() => {
        // Refresh event data periodically or on mount to ensure status is up to date
        // If event is passed, use its ID. If not, we might need ID from elsewhere (e.g. params if we used routing)
        // For now, assume if event is null, we can't do much unless we have an ID. 
        if (event?.id) {
            fetchEventDetails();
        }
    }, [event]);

    useEffect(() => {
        if (currentEvent) {
            Animated.timing(activeStatusAnim, {
                toValue: currentEvent.isActive ? 1 : 0,
                duration: 300,
                useNativeDriver: false,
            }).start();
        }
    }, [currentEvent?.isActive]);

    if (!currentEvent) {
        return (
            <View style={styles.container}>
                <StatusBar style="light" />
                <View style={styles.header}>
                    <TouchableOpacity onPress={onBack} style={styles.backLink}>
                        <Text style={styles.backLinkText}>← Back</Text>
                    </TouchableOpacity>
                    <Text style={styles.headerTitle}>Event Dashboard</Text>
                </View>
                <View style={[styles.content, { justifyContent: 'center', alignItems: 'center' }]}>
                    <Text style={{ color: '#FFF' }}>Loading event details...</Text>
                </View>
            </View>
        );
    }

    const fetchEventDetails = async () => {
        try {
            // Fetch specific event details
            const response = await fetch(`${API_URL}/events/${event.id}`);
            if (response.ok) {
                const data = await response.json();
                setCurrentEvent(data);
            } else {
                // Fallback if specific endpoint doesn't exist
                const listResponse = await fetch(`${API_URL}/events`);
                if (listResponse.ok) {
                    const listData = await listResponse.json();
                    const found = listData.find(e => e.id === event.id);
                    if (found) setCurrentEvent(found);
                }
            }
        } catch (error) {
            console.error('Error fetching event details:', error);
        }
    };

    const handleToggleStatus = async () => {
        try {
            const response = await fetch(`${API_URL}/events/${currentEvent.id}/status`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ isActive: !currentEvent.isActive }),
            });

            if (response.ok) {
                const updatedEvent = await response.json();
                // If backend returns the updated event, use it. Otherwise toggle locally.
                // Assuming backend might just return 200 OK.
                setCurrentEvent(prev => ({ ...prev, isActive: !prev.isActive }));
                Alert.alert('Success', `Event ${!currentEvent.isActive ? 'started' : 'stopped'}`);
            } else {
                Alert.alert('Error', 'Failed to update status');
            }
        } catch (error) {
            console.error('Error updating status:', error);
            Alert.alert('Error', 'Failed to update status');
        }
    };

    const getShareLink = () => {
        if (typeof window !== 'undefined') {
            return `${window.location.origin}/?pin=${currentEvent.id}`;
        }
        return `http://yourapp.com/?pin=${currentEvent.id}`;
    };

    const copyToClipboard = () => {
        const shareLink = getShareLink();
        Clipboard.setString(shareLink);
        if (Platform.OS === 'web') {
            navigator.clipboard.writeText(shareLink).then(() => {
                Alert.alert('Copied', 'Event link copied to clipboard!');
            });
        } else {
            Alert.alert('Copied', 'Event link copied to clipboard!');
        }
    };

    const StatusIndicator = () => {
        const backgroundColor = activeStatusAnim.interpolate({
            inputRange: [0, 1],
            outputRange: ['rgba(255, 69, 58, 0.2)', 'rgba(48, 209, 88, 0.2)']
        });
        const textColor = activeStatusAnim.interpolate({
            inputRange: [0, 1],
            outputRange: ['#FF453A', '#30D158']
        });

        return (
            <Animated.View style={[styles.statusBadge, { backgroundColor }]}>
                <Animated.Text style={[styles.statusText, { color: textColor }]}>
                    {currentEvent.isActive ? 'ACTIVE - RUNNING' : 'INACTIVE - STOPPED'}
                </Animated.Text>
            </Animated.View>
        );
    };

    return (
        <View style={styles.container}>
            <StatusBar style="light" />

            <View style={styles.header}>
                <TouchableOpacity onPress={onBack} style={styles.backLink}>
                    <Text style={styles.backLinkText}>← Back to Dashboard</Text>
                </TouchableOpacity>
                <Text style={styles.headerTitle}>Event Dashboard</Text>
                <Text style={styles.headerSubtitle}>{currentEvent.name}</Text>
            </View>

            <ScrollView style={styles.content} contentContainerStyle={styles.scrollContent}>

                {/* 1. Event Info Card */}
                <View style={styles.card}>
                    <View style={styles.cardHeader}>
                        <Text style={styles.cardTitle}>Event Details</Text>
                        <StatusIndicator />
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Event Name:</Text>
                        <Text style={styles.value}>{currentEvent.name}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Description:</Text>
                        <Text style={styles.value}>{currentEvent.description}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>PIN Code:</Text>
                        <View style={styles.pinContainer}>
                            <Text style={styles.pinText}>{currentEvent.id}</Text>
                        </View>
                    </View>
                </View>

                {/* 2. Share Access Card (New) */}
                <View style={styles.card}>
                    <Text style={styles.cardTitle}>Share Access</Text>
                    <Text style={styles.cardDescription}>
                        Share this QR code or link with participants to let them join the event directly.
                    </Text>

                    <View style={styles.qrCodeContainer}>
                        {Platform.OS === 'web' ? (
                            <div style={{ background: 'white', padding: '16px', borderRadius: '16px' }}>
                                <QRCode
                                    value={getShareLink()}
                                    size={200}
                                    style={{ height: "auto", maxWidth: "100%", width: "100%" }}
                                    viewBox={`0 0 256 256`}
                                />
                            </div>
                        ) : (
                            <View style={styles.qrPlaceholder}>
                                <Text style={styles.qrPlaceholderText}>QR Code available on Web</Text>
                            </View>
                        )}
                    </View>

                    <TouchableOpacity style={styles.copyButton} onPress={copyToClipboard}>
                        <Text style={styles.copyButtonText}>Copy Shareable Link</Text>
                    </TouchableOpacity>
                </View>

                {/* 3. Control Card */}
                <View style={styles.card}>
                    <Text style={styles.cardTitle}>Event Controls</Text>
                    <Text style={styles.cardDescription}>
                        Manage the lifecycle of the event. Stopping the event will disconnect all users.
                    </Text>

                    <TouchableOpacity
                        style={[styles.actionButton, currentEvent.isActive ? styles.stopButton : styles.startButton]}
                        onPress={handleToggleStatus}
                    >
                        <Text style={styles.actionButtonText}>
                            {currentEvent.isActive ? 'STOP EVENT' : 'START EVENT'}
                        </Text>
                    </TouchableOpacity>
                </View>

                {/* Placeholder for more cards */}
                <View style={[styles.card, styles.statsCard]}>
                    <Text style={styles.cardTitle}>Quick Stats</Text>
                    <View style={styles.statsGrid}>
                        <View style={styles.statItem}>
                            <Text style={styles.statNumber}>0</Text>
                            <Text style={styles.statLabel}>Active Users</Text>
                        </View>
                        <View style={styles.statItem}>
                            <Text style={styles.statNumber}>00:00</Text>
                            <Text style={styles.statLabel}>Duration</Text>
                        </View>
                    </View>
                </View>

            </ScrollView>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212',
    },
    header: {
        paddingTop: 60,
        paddingBottom: 20,
        paddingHorizontal: 24,
        backgroundColor: '#1E1E1E',
        borderBottomWidth: 1,
        borderBottomColor: '#2C2C2C',
    },
    backLink: {
        marginBottom: 10,
    },
    backLinkText: {
        color: '#007AFF',
        fontSize: 16,
    },
    headerTitle: {
        fontSize: 14,
        color: '#888',
        textTransform: 'uppercase',
        letterSpacing: 1,
        fontWeight: '600',
    },
    headerSubtitle: {
        fontSize: 28,
        fontWeight: '800',
        color: '#FFF',
        marginTop: 4,
    },
    content: {
        flex: 1,
    },
    scrollContent: {
        padding: 24,
        paddingBottom: 40,
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 24,
        alignItems: 'flex-start',
        maxWidth: 1600,
        width: '100%',
        alignSelf: 'center',
    },
    card: {
        backgroundColor: '#1E1E1E',
        borderRadius: 16,
        padding: 24,
        marginBottom: 0,
        borderWidth: 1,
        borderColor: '#2C2C2C',
        minWidth: 350,
        flex: 1,
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 20,
    },
    cardTitle: {
        fontSize: 20,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 12,
    },
    cardDescription: {
        fontSize: 14,
        color: '#AAA',
        marginBottom: 20,
        lineHeight: 20,
    },
    infoRow: {
        marginBottom: 16,
    },
    label: {
        fontSize: 12,
        color: '#888',
        marginBottom: 4,
        textTransform: 'uppercase',
    },
    value: {
        fontSize: 16,
        color: '#FFF',
    },
    pinContainer: {
        backgroundColor: '#000',
        padding: 10,
        borderRadius: 8,
        alignSelf: 'flex-start',
    },
    pinText: {
        fontSize: 24,
        color: '#FFF',
        fontWeight: 'bold',
        fontFamily: Platform.OS === 'ios' ? 'Courier' : 'monospace',
        letterSpacing: 2,
    },
    statusBadge: {
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 8,
    },
    statusText: {
        fontSize: 12,
        fontWeight: '700',
    },
    copyButton: {
        marginTop: 20,
        backgroundColor: '#333',
        paddingVertical: 14,
        borderRadius: 8,
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#444',
    },
    copyButtonText: {
        color: '#FFF',
        fontWeight: '600',
        fontSize: 14,
    },
    actionButton: {
        paddingVertical: 16,
        borderRadius: 8,
        alignItems: 'center',
    },
    startButton: {
        backgroundColor: '#30D158',
    },
    stopButton: {
        backgroundColor: '#FF453A',
    },
    actionButtonText: {
        color: '#FFF',
        fontWeight: '700',
        fontSize: 16,
    },
    statsCard: {
    },
    statsGrid: {
        flexDirection: 'row',
        justifyContent: 'space-between',
    },
    statItem: {
        flex: 1,
        backgroundColor: '#121212',
        marginRight: 10,
        padding: 16,
        borderRadius: 12,
        alignItems: 'center',
    },
    statNumber: {
        fontSize: 24,
        fontWeight: '700',
        color: '#FFF',
        marginBottom: 4,
    },
    statLabel: {
        fontSize: 12,
        color: '#666',
    },
    qrCodeContainer: {
        alignItems: 'center',
        justifyContent: 'center',
        marginVertical: 10,
        padding: 10,
        backgroundColor: '#2C2C2C',
        borderRadius: 12,
    },
    qrPlaceholder: {
        height: 200,
        width: 200,
        backgroundColor: '#333',
        justifyContent: 'center',
        alignItems: 'center',
        borderRadius: 12,
    },
    qrPlaceholderText: {
        color: '#AAA',
        textAlign: 'center',
    },
});
